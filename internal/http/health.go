package http

import (
	"net/http"
)

// healthHandler 是探活接口，检查数据库连通性。
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	// S1 降级启动时 store 可能为 nil（DB 未接入），此时探活返回 degraded 而非直接不可用。
	if s.store == nil {
		ok(w, r, map[string]string{"status": "degraded", "db": "unavailable"})
		return
	}
	if err := s.store.Ping(r.Context()); err != nil {
		fail(w, r, http.StatusServiceUnavailable, CodeDBUnavailable, "数据库不可用")
		return
	}
	ok(w, r, map[string]string{"status": "healthy"})
}
