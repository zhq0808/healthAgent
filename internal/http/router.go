// Package http 提供 HTTP 接口层：路由、中间件、DTO、统一响应。
package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"healthAgent/internal/llm"
)

// Server 持有 HTTP 层依赖，并挂载路由。
type Server struct {
	llm    *llm.DeepSeekClient
	log    *slog.Logger
	engine *gin.Engine
}

// NewServer 构建 HTTP Server 并注册路由与中间件。
func NewServer(llmClient *llm.DeepSeekClient, log *slog.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		llm:    llmClient,
		log:    log,
		engine: gin.New(), // 不用 gin.Default()，用我们自己的中间件（日志/recover）
	}
	s.routes()
	return s
}

// maxBodyBytes 是默认请求体大小上限（2MB）。
// 文本录入足够；语音/文件上传接口后续可单独放宽。
const maxBodyBytes = 2 << 20

// routes 注册中间件与路由。P0 只挂探活；业务路由在 Phase 1/2 逐步加入。
func (s *Server) routes() {
	s.engine.Use(traceMiddleware())
	s.engine.Use(recoverMiddleware(s.log))
	s.engine.Use(accessLogMiddleware(s.log))
	s.engine.Use(bodyLimitMiddleware(maxBodyBytes))

	s.engine.NoRoute(func(c *gin.Context) {
		fail(c, http.StatusNotFound, CodeNotFound, "接口不存在")
	})
	s.engine.NoMethod(func(c *gin.Context) {
		fail(c, http.StatusMethodNotAllowed, CodeMethodNA, "方法不允许")
	})

	s.engine.GET("/health", s.healthHandler)

	// 业务路由。竖切片逐步加入。
	v1 := s.engine.Group("/api/v1")
	{
		v1.POST("/chat", s.chatHandler) // S1 对话
	}
}

// Handler 返回底层 http.Handler，供 http.Server 使用。
func (s *Server) Handler() http.Handler {
	return s.engine
}
