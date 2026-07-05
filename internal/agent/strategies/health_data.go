package strategies

import (
	"context"

	"healthAgent/internal/llm"
)

// HealthData 处理「录入身体/体检指标」意图。
type HealthData struct {
	client *llm.DeepSeekClient
}

// NewHealthData 构建体检指标录入策略。
func NewHealthData(client *llm.DeepSeekClient) *HealthData {
	return &HealthData{client: client}
}

// Handle 目前为骨架，暂时走通用对话。
// TODO(S2+): 调 prompt.HealthExtract 抽取结构化指标 → 落库 → 返回确认。
func (h *HealthData) Handle(ctx context.Context, input string) (string, error) {
	return genericChat(ctx, h.client, input)
}
