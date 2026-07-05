// Package strategies 实现各意图的具体处理器（策略模式）。
// 每个策略只处理一类意图，由 agent.Dispatcher 按意图选择调用。
package strategies

import (
	"context"

	"healthAgent/internal/llm"
	"healthAgent/internal/prompt"
)

// Strategy 处理某一类意图，返回给用户的回复文本。
type Strategy interface {
	Handle(ctx context.Context, input string) (string, error)
}

// genericChat 走通用健康助手对话（自带系统人设），供尚未实现专属逻辑的策略占位复用。
func genericChat(ctx context.Context, client *llm.DeepSeekClient, input string) (string, error) {
	return client.Complete(ctx, []llm.Message{
		{Role: "system", Content: prompt.System()},
		{Role: "user", Content: input},
	})
}
