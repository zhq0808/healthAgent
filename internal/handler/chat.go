package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"healthAgent/internal/llm"
	"healthAgent/internal/service"
)

// chatRequest 是对话接口的请求体。
type chatRequest struct {
	SessionID       string `json:"session_id"`
	ClientMessageID string `json:"client_message_id"`
	Message         string `json:"message"`
}

var clientMessageIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

const recentHistoryLimit = 20

// chatStreamHandler 处理一条用户消息，以 SSE 流式把模型回复逐段推给前端。
// Content-Type: text/event-stream，每收到一段增量就 `data: {json}\n\n` 刷一帧，
// 结束发一帧 `event: done`。客户端断开时：ctx 取消 + Write 报错双重兑底停止。
func (s *Server) chatStreamHandler(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ClientMessageID = strings.ToLower(strings.TrimSpace(req.ClientMessageID))
	if !validSessionID(req.SessionID) {
		fail(c, http.StatusBadRequest, CodeBadRequest, "会话ID格式错误")
		return
	}
	if !clientMessageIDPattern.MatchString(req.ClientMessageID) {
		fail(c, http.StatusBadRequest, CodeBadRequest, "客户端消息ID格式错误")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		fail(c, http.StatusBadRequest, CodeBadRequest, "消息不能为空")
		return
	}
	userID, ok := UserIDFromContext(c.Request.Context())
	if !ok {
		fail(c, http.StatusUnauthorized, CodeUnauthorized, "请先建立访客身份")
		return
	}
	if err := s.sessions.RequireOwnedActive(c.Request.Context(), userID, req.SessionID); err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			fail(c, http.StatusNotFound, CodeNotFound, "会话不存在")
			return
		}
		s.log.Error("校验会话归属失败",
			"trace_id", TraceIDFromContext(c.Request.Context()),
			"error", err,
		)
		fail(c, http.StatusInternalServerError, CodeInternal, "会话服务暂时不可用")
		return
	}

	messageResult, err := s.messages.AppendUserMessage(c.Request.Context(), service.AppendUserMessageRequest{
		UserID:          userID,
		SessionID:       req.SessionID,
		ClientMessageID: req.ClientMessageID,
		Content:         strings.TrimSpace(req.Message),
		TraceID:         TraceIDFromContext(c.Request.Context()),
	})
	if errors.Is(err, service.ErrClientMessageConflict) {
		fail(c, http.StatusConflict, CodeConflict, "客户端消息ID已用于其他内容")
		return
	}
	if errors.Is(err, service.ErrSessionNotFound) {
		fail(c, http.StatusNotFound, CodeNotFound, "会话不存在")
		return
	}
	if err != nil {
		s.log.Error("用户消息落库失败",
			"trace_id", TraceIDFromContext(c.Request.Context()),
			"error", err,
		)
		fail(c, http.StatusInternalServerError, CodeInternal, "消息服务暂时不可用")
		return
	}
	if !messageResult.Created {
		fail(c, http.StatusConflict, CodeConflict, "消息已提交，请勿重复发送")
		return
	}

	// 占用本次 turn 的租约：防止同一 Session 被两个请求同时处理（重复点击/前端重试）。
	// 占不到就是别的请求正占着（还没过期），直接 409，不再往下走。
	leaseResult, err := s.turnLeases.Acquire(c.Request.Context(), service.AcquireTurnLeaseRequest{
		UserID:          userID,
		SessionID:       req.SessionID,
		ClientMessageID: req.ClientMessageID,
	})
	if errors.Is(err, service.ErrTurnLeaseConflict) {
		fail(c, http.StatusConflict, CodeConflict, "该会话正在处理上一条消息，请稍后重试")
		return
	}
	if err != nil {
		s.log.Error("获取 turn 租约失败",
			"trace_id", TraceIDFromContext(c.Request.Context()),
			"error", err,
		)
		fail(c, http.StatusInternalServerError, CodeInternal, "对话服务暂时不可用")
		return
	}
	if !leaseResult.Acquired {
		// 走到这里说明这个全新的 client_message_id 竟然已经有历史终态租约记录，属于异常状态，
		// 按冲突处理更安全：不重复触发 LLM 调用。
		s.log.Error("turn 租约状态异常：新消息命中历史终态记录",
			"trace_id", TraceIDFromContext(c.Request.Context()),
			"lease_status", leaseResult.Lease.Status,
		)
		fail(c, http.StatusConflict, CodeConflict, "消息已处理，请勿重复发送")
		return
	}

	// 无论后面正常结束、LLM 报错还是落库失败，都必须释放租约，否则这个 Session 会一直
	// 被锁到租约自然过期。用 defer + 独立 context：客户端断开/请求超时时 c.Request.Context()
	// 可能已经被取消，释放这个"短事务写"不应该跟着客户端的生命周期一起被取消。
	leaseStatus := service.TurnLeaseFailed
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer releaseCancel()
		if releaseErr := s.turnLeases.Release(releaseCtx, service.ReleaseTurnLeaseRequest{
			UserID:          userID,
			SessionID:       req.SessionID,
			ClientMessageID: req.ClientMessageID,
			Status:          leaseStatus,
		}); releaseErr != nil {
			s.log.Error("释放 turn 租约失败",
				"trace_id", TraceIDFromContext(c.Request.Context()),
				"error", releaseErr,
			)
		}
	}()

	history, err := s.messages.LoadRecent(c.Request.Context(), userID, req.SessionID, recentHistoryLimit)
	if err != nil {
		s.log.Error("读取对话历史失败",
			"trace_id", TraceIDFromContext(c.Request.Context()),
			"error", err,
		)
		fail(c, http.StatusInternalServerError, CodeInternal, "读取对话历史失败")
		return
	}

	// http.Flusher：每写一帧就 flush 到网络，否则会被缓冲攺着一次性发，流式就失意了。
	// gin.ResponseWriter 本身实现了 http.Flusher，这里断言一下更显意图。
	flusher, canFlush := c.Writer.(http.Flusher)
	if !canFlush {
		fail(c, http.StatusInternalServerError, CodeInternal, "服务器不支持流式输出")
		return
	}

	// SSE 响应头（必须在写任何 body 前设好）。
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 关闭 Nginx 缓冲，保证逐帧下发

	ctx, cancel := context.WithTimeout(c.Request.Context(), s.chat.Timeout())
	defer cancel()

	// writeSSE 写一帧并立即 flush；写失败（客户端断开）返回 error 以便上游停止。
	writeSSE := func(event, data string) error {
		var b strings.Builder
		if event != "" {
			b.WriteString("event: ")
			b.WriteString(event)
			b.WriteByte('\n')
		}
		b.WriteString("data: ")
		b.WriteString(data)
		b.WriteString("\n\n")
		if _, err := io.WriteString(c.Writer, b.String()); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	// 把一段增量包成 JSON 再下发，避免文本里的换行破坏 SSE 帧结构。
	var assistantContent strings.Builder
	sendDelta := func(delta string) error {
		assistantContent.WriteString(delta)
		payload, _ := json.Marshal(map[string]string{"delta": delta})
		return writeSSE("", string(payload))
	}

	err = s.chat.Stream(ctx, history, sendDelta)
	if err != nil {
		// 未配置 Key：不算服务故障，当作一段普通回复流出去，方便本地先跑通链路。
		if errors.Is(err, llm.ErrNotConfigured) {
			_ = sendDelta("（未配置大模型 API Key，请在 .env 填入 DEEPSEEK_API_KEY 后重试）")
			_ = writeSSE("done", "{}")
			leaseStatus = service.TurnLeaseCompleted
			return
		}
		s.log.Error("流式对话调用失败",
			"trace_id", TraceIDFromContext(c.Request.Context()),
			"error", err,
		)
		// 已经开始流（header 无法再改成 500），用 error 事件通知前端。
		_ = writeSSE("error", `{"message":"对话服务暂时不可用"}`)
		return
	}

	if _, err := s.messages.AppendAssistantMessage(c.Request.Context(), service.AppendAssistantMessageRequest{
		UserID:    userID,
		SessionID: req.SessionID,
		Content:   assistantContent.String(),
		TraceID:   TraceIDFromContext(c.Request.Context()),
	}); err != nil {
		s.log.Error("assistant 消息落库失败",
			"trace_id", TraceIDFromContext(c.Request.Context()),
			"error", err,
		)
		_ = writeSSE("error", `{"message":"回复保存失败，请稍后重试"}`)
		return
	}

	_ = writeSSE("done", "{}")
	leaseStatus = service.TurnLeaseCompleted
}
