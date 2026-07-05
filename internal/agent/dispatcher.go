package agent

import (
	"context"

	"healthAgent/internal/agent/strategies"
	"healthAgent/internal/llm"
	"healthAgent/internal/models"
	"healthAgent/internal/prompt"
)

// Dispatcher 按意图把请求分发给对应的策略处理器。
type Dispatcher struct {
	client   *llm.DeepSeekClient
	handlers map[string]strategies.Strategy
}

// NewDispatcher 构建分发器并注册各意图的策略处理器。
func NewDispatcher(client *llm.DeepSeekClient) *Dispatcher {
	return &Dispatcher{
		client: client,
		handlers: map[string]strategies.Strategy{
			models.IntentRecordHealthData: strategies.NewHealthData(client),
			models.IntentRecordFood:       strategies.NewFoodRecord(client),
			models.IntentAskDietAdvice:    strategies.NewDietAdvice(client),
		},
	}
}

// Dispatch 选择意图对应的策略处理；other_chat 或未知意图走通用对话兜底。
func (d *Dispatcher) Dispatch(ctx context.Context, intent, input string) (string, error) {
	if s, ok := d.handlers[intent]; ok {
		return s.Handle(ctx, input)
	}
	return d.client.Complete(ctx, []llm.Message{
		{Role: "system", Content: prompt.System()},
		{Role: "user", Content: input},
	})
}
