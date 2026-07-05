package handler

import (
	"github.com/gin-gonic/gin"
)

// healthHandler 是探活接口。
// 骨架阶段不依赖任何外部存储，直接返回健康；接入 DB 后再在此加依赖探活。
func (s *Server) healthHandler(c *gin.Context) {
	ok(c, map[string]string{"status": "healthy"})
}
