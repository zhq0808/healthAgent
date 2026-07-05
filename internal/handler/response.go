package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 是全站统一响应结构。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// writeJSON 写入统一格式的 JSON 响应，并携带 trace_id。
func writeJSON(c *gin.Context, httpStatus, code int, message string, data any) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
		TraceID: TraceIDFromContext(c.Request.Context()),
	})
}

// ok 返回成功响应。
func ok(c *gin.Context, data any) {
	writeJSON(c, http.StatusOK, CodeOK, "ok", data)
}

// fail 返回错误响应。
func fail(c *gin.Context, httpStatus, code int, message string) {
	writeJSON(c, httpStatus, code, message, nil)
}
