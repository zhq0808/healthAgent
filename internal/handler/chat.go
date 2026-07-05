package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"healthAgent/internal/models"
)

// chatRequest 是对话接口的请求体。
type chatRequest struct {
	Message string `json:"message"`
}

// chatReply 是对话接口的响应数据。
type chatReply struct {
	Reply  string `json:"reply"`
	Intent string `json:"intent,omitempty"`
}

// chatHandler 处理一条用户消息：校验入参 → 交给 Agent 编排 → 返回回复与意图。
func (s *Server) chatHandler(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}
	if req.Message == "" {
		fail(c, http.StatusBadRequest, CodeBadRequest, "消息不能为空")
		return
	}

	traceID := TraceIDFromContext(c.Request.Context())
	reply, intent := s.agent.Handle(c.Request.Context(), req.Message, traceID, string(models.DefaultUserID))

	ok(c, chatReply{Reply: reply, Intent: intent})
}
