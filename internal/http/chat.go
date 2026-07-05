package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"healthAgent/internal/intent"
	"healthAgent/internal/model"
)

// chatRequest 是 POST /api/v1/chat 的请求体。
type chatRequest struct {
	Message string `json:"message"`
}

// cardPayload 是回复可选附带的结构化卡片。前端按 Type 渲染对应卡片（数据暂由前端 mock，属 S6+）。
type cardPayload struct {
	Type string `json:"type"`
}

// chatReply 是对话回复体。Reply 为文本回复；Intent 为识别出的意图；Card 可选。
type chatReply struct {
	Reply  string       `json:"reply"`
	Intent string       `json:"intent,omitempty"`
	Card   *cardPayload `json:"card,omitempty"`
}

// chatHandler 处理对话请求：校验入参 → 意图识别 → 分流 → 调 LLM → 返回。
func (s *Server) chatHandler(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeBadRequest, "请求体解析失败")
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		fail(c, http.StatusBadRequest, CodeBadRequest, "message 不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), s.llm.Timeout())
	defer cancel()

	traceID := TraceIDFromContext(c.Request.Context())
	userID := string(model.DefaultUserID)

	// 意图识别（优先 LLM，失败走关键词兜底）；后续按意图分流到各处理链路。
	it := s.classifyIntent(ctx, message, traceID, userID)

	reply, err := s.llm.Chat(ctx, message)
	if err != nil {
		s.log.Warn("调用大模型失败，降级返回兜底回复",
			"trace_id", traceID, "user_id", userID, "error", err)
		ok(c, chatReply{Reply: "抱歉，我现在有点忙，稍后再问我一次好吗？"})
		return
	}
	ok(c, chatReply{Reply: reply, Intent: it})
}

// classifyIntent 优先用 LLM 识别意图，失败时降级到关键词兜底。
// 两条路径都带 trace_id/user_id 日志，便于按请求链路定位问题。
func (s *Server) classifyIntent(ctx context.Context, message, traceID, userID string) string {
	res, err := s.llm.ClassifyIntent(ctx, message)
	if err != nil {
		it := intent.Fallback(message)
		s.log.Warn("意图识别降级：LLM 失败，走关键词兜底",
			"trace_id", traceID, "user_id", userID, "intent", it, "error", err)
		return it
	}
	s.log.Info("意图识别成功",
		"trace_id", traceID, "user_id", userID,
		"intent", res.Intent, "confidence", res.Confidence)
	return res.Intent
}
