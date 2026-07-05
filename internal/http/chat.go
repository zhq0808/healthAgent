package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

// chatRequest 是 POST /api/v1/chat 的请求体。
type chatRequest struct {
	Message string `json:"message"`
}

// chatReply 是对话回复体。S1-a 先返回写死内容；S1-b 接入 DeepSeek 后返回真回复。
type chatReply struct {
	Reply string `json:"reply"`
}

// chatHandler 处理对话请求。
//
// S1-a（当前）：只打通「前端 → 后端 → 前端」这条管道，回写死内容，不碰 DB、不调 LLM。
// S1-b（下一步，你来写）：把写死回复换成真调 DeepSeek —— 读 .env 的 key、带 context 超时、失败降级。
func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, r, http.StatusBadRequest, CodeBadRequest, "请求体解析失败")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		fail(w, r, http.StatusBadRequest, CodeBadRequest, "message 不能为空")
		return
	}

	// S1-a：写死回复，仅用于验证管道。把用户输入回显出来，方便一眼确认链路通。
	ok(w, r, chatReply{Reply: "pong：你刚才说的是「" + req.Message + "」"})
}
