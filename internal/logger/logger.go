// Package logger 提供结构化日志。
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// TraceIDKey 是全局唯一的 Context Key，供中间件和日志包共用
type contextKey string

const TraceIDKey contextKey = "trace_id"

// ContextHandler 包装原生的 slog.Handler，自动提取 Context 中的元数据
type ContextHandler struct {
	slog.Handler
}

// Handle 拦截每一条日志记录，自动注入 trace_id
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		// 将 trace_id 追加到日志属性中
		r.AddAttrs(slog.String("trace_id", traceID))
	}
	return h.Handler.Handle(ctx, r)
}

// New 创建一个注入了 ContextHandler 的结构化 logger
func New(level string, debug bool) *slog.Logger {
	lv := parseLevel(level)
	if debug {
		lv = slog.LevelDebug
	}

	// 1. 创建基础的 JSON Handler，并保留源码位置输出
	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     lv,
		AddSource: true,
	})

	// 2. 用自定义的 ContextHandler 包装它
	h := &ContextHandler{Handler: baseHandler}

	return slog.New(h)
}

// parseLevel 把字符串日志级别转为 slog.Level，未知值兜底为 info。
func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Redact 对敏感字符串脱敏，仅保留是否有值的信息，绝不输出原文。
// 用于必须记录“某敏感字段存在”但不能泄露内容的场景。
func Redact(s string) string {
	if s == "" {
		return ""
	}
	return "[REDACTED]"
}
