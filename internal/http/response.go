package http

import (
	"encoding/json"
	"net/http"
)

// Response 是全站统一响应结构。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// writeJSON 写入统一格式的 JSON 响应，并携带 trace_id。
func writeJSON(w http.ResponseWriter, r *http.Request, httpStatus, code int, message string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
		Data:    data,
		TraceID: TraceIDFromContext(r.Context()),
	})
}

// ok 返回成功响应。
func ok(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, r, http.StatusOK, 0, "ok", data)
}

// fail 返回错误响应。
func fail(w http.ResponseWriter, r *http.Request, httpStatus, code int, message string) {
	writeJSON(w, r, httpStatus, code, message, nil)
}
