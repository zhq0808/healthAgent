package http

import (
	"net/http"
)

// healthHandler 是探活接口，检查数据库连通性。
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		fail(w, r, http.StatusServiceUnavailable, CodeDBUnavailable, "数据库不可用")
		return
	}
	ok(w, r, map[string]string{"status": "healthy"})
}
