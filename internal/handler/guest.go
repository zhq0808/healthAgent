package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// guestReply 是 Guest 创建接口的响应数据。
type guestReply struct {
	UserID  string `json:"user_id"`
	Created bool   `json:"created"`
}

// guestHandler 恢复当前设备的 Guest 身份；没有有效凭证时原子创建用户和凭证。
func (s *Server) guestHandler(c *gin.Context) {
	rawToken, _ := c.Cookie(s.identityConfig.GuestCookieName)
	identity, err := s.identity.EnsureGuest(c.Request.Context(), rawToken)
	if err != nil {
		s.log.Error("创建或恢复 Guest 身份失败", "trace_id", TraceIDFromContext(c.Request.Context()), "error", err)
		fail(c, http.StatusInternalServerError, CodeInternal, "创建用户失败")
		return
	}

	if identity.Token != "" {
		maxAge := int(time.Until(identity.ExpiresAt).Seconds())
		if maxAge < 1 {
			maxAge = 1
		}
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     s.identityConfig.GuestCookieName,
			Value:    identity.Token,
			Path:     "/",
			Expires:  identity.ExpiresAt,
			MaxAge:   maxAge,
			HttpOnly: true,
			Secure:   s.identityConfig.CookieSecure,
			SameSite: http.SameSiteLaxMode,
		})
	}

	c.Header("Cache-Control", "no-store")
	ok(c, guestReply{UserID: identity.UserID, Created: identity.Created})
}
