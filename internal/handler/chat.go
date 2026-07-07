package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"healthAgent/internal/llm"
)

// systemPrompt 是健康管家的系统人设。基础对话阶段先内联，
// 后续接入意图识别/推荐策略时再迁到独立的 prompt 层做拼装。
const systemPrompt = `你是一个专业、亲切的个人 AI 健康管家，服务于关注体检异常指标与日常饮食的用户。
用简洁、口语化的中文回答，直奔主题，不堆砌套话。
涉及医疗建议时，提醒用户异常情况应及时就医，你不替代专业医生诊断。`

// chatRequest 是对话接口的请求体。
//
// History 是之前若干轮的对话（不含本条 Message），由前端维护并每次带上，
// 让后端保持无状态。角色只认 user/assistant，system 由后端自己拼，不信任前端传入。
type chatRequest struct {
	Message string        `json:"message"`
	History []llm.Message `json:"history,omitempty"`
}

// maxHistoryMessages 是带入模型的历史条数上限：只保留最近 N 条，
// 防止多轮后 prompt 无限膨胀（token 成本 + 超模型上下文窗口）。
const maxHistoryMessages = 20

// buildMessages 把「系统人设 + 历史 + 本条用户消息」组装成发给模型的完整消息序列。
//
// 两处安全/健壮性处理（都是系统边界该做的校验）：
//  1. 只信任 user/assistant 角色，过滤掉前端可能注入的 system，避免人设被篡改；
//  2. 历史超限时只截取最近 maxHistoryMessages 条（滑动窗口）。
func buildMessages(req chatRequest) []llm.Message {
	history := req.History
	if len(history) > maxHistoryMessages {
		history = history[len(history)-maxHistoryMessages:]
	}

	messages := make([]llm.Message, 0, len(history)+2)
	messages = append(messages, llm.Message{Role: "system", Content: systemPrompt})
	for _, m := range history {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		messages = append(messages, m)
	}
	messages = append(messages, llm.Message{Role: "user", Content: req.Message})
	return messages
}

// chatReply 是对话接口的响应数据。
type chatReply struct {
	Reply string `json:"reply"`
}

// chatHandler 处理一条用户消息：校验入参 → 拼系统人设 → 调大模型 → 返回回复。
func (s *Server) chatHandler(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		fail(c, http.StatusBadRequest, CodeBadRequest, "消息不能为空")
		return
	}

	// 从请求 context 派生带 deadline 的 context：客户端断开或超时时能主动取消底层 HTTP 调用。
	ctx, cancel := context.WithTimeout(c.Request.Context(), s.llm.Timeout())
	defer cancel()

	messages := buildMessages(req)

	reply, err := s.llm.Complete(ctx, messages)
	if err != nil {
		// 未配置 Key：不算服务故障，返回友好提示，方便本地先跑通链路。
		if errors.Is(err, llm.ErrNotConfigured) {
			ok(c, chatReply{Reply: "（未配置大模型 API Key，请在 .env 填入 DEEPSEEK_API_KEY 后重试）"})
			return
		}
		s.log.Error("对话调用失败",
			"trace_id", TraceIDFromContext(c.Request.Context()),
			"error", err,
		)
		fail(c, http.StatusBadGateway, CodeUpstream, "对话服务暂时不可用，请稍后重试")
		return
	}

	ok(c, chatReply{Reply: reply})
}

// chatStreamHandler 处理一条用户消息，以 SSE 流式把模型回复逐段推给前端。
//
// 与 chatHandler 的区别：不再等全文再一次性返回统一 JSON，而是
// Content-Type: text/event-stream，每收到一段增量就 `data: {json}\n\n` 刷一帧，
// 结束发一帧 `event: done`。客户端断开时：ctx 取消 + Write 报错双重兑底停止。
func (s *Server) chatStreamHandler(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		fail(c, http.StatusBadRequest, CodeBadRequest, "消息不能为空")
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), s.llm.Timeout())
	defer cancel()

	messages := buildMessages(req)

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
	sendDelta := func(delta string) error {
		payload, _ := json.Marshal(map[string]string{"delta": delta})
		return writeSSE("", string(payload))
	}

	err := s.llm.Stream(ctx, messages, sendDelta)
	if err != nil {
		// 未配置 Key：不算服务故障，当作一段普通回复流出去，方便本地先跑通链路。
		if errors.Is(err, llm.ErrNotConfigured) {
			_ = sendDelta("（未配置大模型 API Key，请在 .env 填入 DEEPSEEK_API_KEY 后重试）")
			_ = writeSSE("done", "{}")
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

	_ = writeSSE("done", "{}")
}
