package strategies

import (
	"context"

	"healthAgent/internal/llm"
)

// FoodRecord 处理「记录饮食」意图。
type FoodRecord struct {
	client *llm.DeepSeekClient
}

// NewFoodRecord 构建饮食记录策略。
func NewFoodRecord(client *llm.DeepSeekClient) *FoodRecord {
	return &FoodRecord{client: client}
}

// Handle 目前为骨架，暂时走通用对话。
// TODO: 调 prompt.FoodRecord 结构化饮食条目 → 落库 → 返回确认。
func (f *FoodRecord) Handle(ctx context.Context, input string) (string, error) {
	return genericChat(ctx, f.client, input)
}
