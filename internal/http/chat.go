package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"healthAgent/internal/intent"
)

// chatRequest 是 POST /api/v1/chat 的请求体。
type chatRequest struct {
	Message string `json:"message"`
}

// cardPayload 是回复可选附带的结构化卡片。前端按 Type 渲染对应卡片（数据暂由前端 mock，属 S6+）。
type cardPayload struct {
	Type string `json:"type"`
}

// chatReply 是对话回复体。Reply 为文本回复；Card 可选，由后端按意图决定是否附带。
type chatReply struct {
	Reply string       `json:"reply"`
	Card  *cardPayload `json:"card,omitempty"`
}

// chatHandler 处理对话请求：校验入参 → 调 DeepSeek → 返回真回复。
//
// S1-b（当前）：真调 DeepSeek。两处关键：
//   - context 超时：从请求 context 派生一个带 deadline 的 context 传给 LLM，
//     DeepSeek 卡住或客户端断开时能主动取消，不把 handler goroutine 挂死。
//   - 错误降级：LLM 失败不把 5xx 甩给前端，返回一句友好兜底，真错误进日志。
func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, r, http.StatusBadRequest, CodeBadRequest, "请求体解析失败")
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		fail(w, r, http.StatusBadRequest, CodeBadRequest, "message 不能为空")
		return
	}

	// 从请求 context 派生带超时的 context：既继承「客户端断开即取消」，又加一层调用超时上限。
	ctx, cancel := context.WithTimeout(r.Context(), s.llm.Timeout())
	defer cancel()

	reply, err := s.llm.Chat(ctx, message)
	if err != nil {
		// 业务层降级：给用户一句友好兜底，真错误只进日志，避免把内部细节暴露给前端。
		s.log.Warn("调用大模型失败，降级返回兜底回复",
			"error", err,
			"trace_id", TraceIDFromContext(r.Context()),
		)
		ok(w, r, chatReply{Reply: "抱歉，我现在有点忙，稍后再问我一次好吗？"})
		return
	}

	// 意图识别在后端完成：命中则附带结构化卡片，前端只负责渲染。
	resp := chatReply{Reply: reply}
	if cardType, matched := intent.Resolve(message); matched {
		resp.Card = &cardPayload{Type: cardType}
	}
	ok(w, r, resp)
}
