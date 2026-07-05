// Package logger 提供结构化日志。
//
// 合规红线：默认不打印体检原文、健康数值、完整 prompt 等敏感信息。
// 只有在 Debug=true（仅限本地）时才允许通过 debug 级别输出敏感原文。
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New 创建一个 JSON 结构化 logger。level 支持 debug/info/warn/error。
func New(level string, debug bool) *slog.Logger {
	lv := parseLevel(level)
	// debug 标志会强制放开到 debug 级别，用于本地排查。
	if debug {
		lv = slog.LevelDebug
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
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
