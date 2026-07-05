package strategies

import (
	"context"

	"healthAgent/internal/llm"
)

// DietAdvice 处理「询问饮食建议」意图。
type DietAdvice struct {
	client *llm.DeepSeekClient
}

// NewDietAdvice 构建饮食建议策略。
func NewDietAdvice(client *llm.DeepSeekClient) *DietAdvice {
	return &DietAdvice{client: client}
}

// Handle 目前为骨架，暂时走通用对话。
// TODO(S6): 组装用户画像（异常指标/过敏/偏好）填入 prompt.DietAdvice 模板 → 生成建议。
func (d *DietAdvice) Handle(ctx context.Context, input string) (string, error) {
	return genericChat(ctx, d.client, input)
}
