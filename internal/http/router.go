// Package http 提供 HTTP 接口层：路由、中间件、DTO、统一响应。
package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"healthAgent/internal/store"
)

// Server 持有 HTTP 层依赖，并挂载路由。
type Server struct {
	store *store.Store
	log   *slog.Logger
	mux   *chi.Mux
}

// NewServer 构建 HTTP Server 并注册路由与中间件。
func NewServer(st *store.Store, log *slog.Logger) *Server {
	s := &Server{
		store: st,
		log:   log,
		mux:   chi.NewRouter(),
	}
	s.routes()
	return s
}

// maxBodyBytes 是默认请求体大小上限（2MB）。
// 文本录入足够；语音/文件上传接口后续可单独放宽。
const maxBodyBytes = 2 << 20

// routes 注册中间件与路由。P0 只挂探活；业务路由在 Phase 1/2 逐步加入。
func (s *Server) routes() {
	s.mux.Use(traceMiddleware)
	s.mux.Use(recoverMiddleware(s.log))
	s.mux.Use(accessLogMiddleware(s.log))
	s.mux.Use(bodyLimitMiddleware(maxBodyBytes))

	s.mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		fail(w, r, http.StatusNotFound, CodeNotFound, "接口不存在")
	})
	s.mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		fail(w, r, http.StatusMethodNotAllowed, CodeMethodNA, "方法不允许")
	})

	s.mux.Get("/health", s.healthHandler)

	// P1/P2 业务路由占位（Phase 1 起实现）：
	// s.mux.Route("/api/v1", func(r chi.Router) {
	//     r.Post("/intake/text", ...)
	//     r.Post("/intake/confirm", ...)
	//     r.Post("/chat", ...)
	//     r.Post("/meals", ...)
	// })
}

// Handler 返回底层 http.Handler，供 http.Server 使用。
func (s *Server) Handler() http.Handler {
	return s.mux
}
