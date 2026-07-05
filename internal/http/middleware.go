package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// ctxKey 是 context 键类型，避免键冲突。
type ctxKey string

const traceIDKey ctxKey = "trace_id"

// newTraceID 生成一个随机的 trace id（16 字节 hex）。
func newTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 极少发生；退化为时间戳，保证不为空。
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}

// TraceIDFromContext 从 context 取出 trace id。
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

// traceMiddleware 为每个请求生成 trace id，写入 context 与响应头。
func traceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid := r.Header.Get("X-Trace-Id")
		if tid == "" {
			tid = newTraceID()
		}
		ctx := context.WithValue(r.Context(), traceIDKey, tid)
		w.Header().Set("X-Trace-Id", tid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// recoverMiddleware 捕获 panic，避免单个请求崩掉整个服务。
// 注意：不记录请求体，避免泄露敏感健康数据。
func recoverMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("请求处理 panic",
						"trace_id", TraceIDFromContext(r.Context()),
						"method", r.Method,
						"path", r.URL.Path,
						"panic", rec,
					)
					fail(w, r, http.StatusInternalServerError, CodeInternal, "内部错误")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder 包装 ResponseWriter 以捕获状态码，供访问日志使用。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// accessLogMiddleware 记录请求耗时与状态码。仅记录安全字段，
// 绝不记录 query/body/header 中可能含健康数据的内容。
func accessLogMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sr, r)
			log.Info("request",
				"trace_id", TraceIDFromContext(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", sr.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

// bodyLimitMiddleware 限制请求体大小，防止大请求/慢连接拖住服务。
func bodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
