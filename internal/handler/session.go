package handler

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

var sessionIDPattern = regexp.MustCompile(`^session_[0-9a-f]{32}$`)

type createSessionReply struct {
	SessionID string `json:"session_id"`
}

func validSessionID(sessionID string) bool {
	return sessionIDPattern.MatchString(sessionID)
}

func (s *Server) createSessionHandler(c *gin.Context) {
	userID, authenticated := UserIDFromContext(c.Request.Context())
	if !authenticated {
		fail(c, http.StatusUnauthorized, CodeUnauthorized, "请先建立访客身份")
		return
	}

	sessionID, err := s.sessions.Create(c.Request.Context(), userID)
	if err != nil {
		s.log.Error("创建会话失败", "trace_id", TraceIDFromContext(c.Request.Context()), "error", err)
		fail(c, http.StatusInternalServerError, CodeInternal, "创建会话失败")
		return
	}
	ok(c, createSessionReply{SessionID: sessionID})
}
