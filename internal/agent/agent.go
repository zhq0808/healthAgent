// Package agent 编排意图分析与策略分发，是对话的核心大脑。
package agent

import (
	"context"
	"log/slog"

	"healthAgent/internal/llm"
)

// Agent 组合意图分析器与策略分发器，对外暴露单一入口 Handle。
type Agent struct {
	client     *llm.DeepSeekClient
	analyzer   *Analyzer
	dispatcher *Dispatcher
	log        *slog.Logger
}

// New 构建 Agent。
func New(client *llm.DeepSeekClient, log *slog.Logger) *Agent {
	return &Agent{
		client:     client,
		analyzer:   NewAnalyzer(client, log),
		dispatcher: NewDispatcher(client),
		log:        log,
	}
}

// Handle 处理一条用户消息：意图识别 → 分流 → 生成回复。
// 返回回复文本与识别出的意图。内部自带调用超时。
func (a *Agent) Handle(ctx context.Context, message, traceID, userID string) (reply, intent string) {
	ctx, cancel := context.WithTimeout(ctx, a.client.Timeout())
	defer cancel()

	res := a.analyzer.Analyze(ctx, message, traceID, userID)

	reply, err := a.dispatcher.Dispatch(ctx, res.Intent, message)
	if err != nil {
		a.log.Warn("对话处理失败，降级返回兜底回复",
			"trace_id", traceID, "user_id", userID, "intent", res.Intent, "error", err)
		return "抱歉，我现在有点忙，稍后再问我一次好吗？", res.Intent
	}
	return reply, res.Intent
}
