// Package intent 定义对话意图，并提供 LLM 不可用时的关键词兜底分类。
package intent

import "strings"

// 意图类型。值必须与 prompt/intent.md 的分类名一致（LLM 按此返回）。
const (
	RecordHealthData = "record_health_data" // 录入身体/体检指标
	RecordFood       = "record_food"        // 记录饮食
	AskDietAdvice    = "ask_diet_advice"    // 询问饮食建议
	OtherChat        = "other_chat"         // 其它闲聊/健康疑问
)

type rule struct {
	intent  string
	phrases []string
}

// 关键词兜底规则：仅在 LLM 分类不可用时使用。
// 用完整短语而非单字，避免「吃」「饭」等高频词误命中（BUG-002）。
var fallbackRules = []rule{
	{RecordHealthData, []string{"血糖", "血压", "血脂", "尿酸", "胆固醇", "甘油三酯", "体重", "体脂", "身高"}},
	{RecordFood, []string{"我吃了", "刚吃了", "喝了", "早餐吃", "午餐吃", "晚餐吃"}},
	{AskDietAdvice, []string{"吃什么", "吃啥", "吃点什么", "膳食", "食谱", "今天吃", "推荐菜", "能吃"}},
}

// Fallback 用关键词匹配兜底分类；兜不中返回 OtherChat。
func Fallback(message string) string {
	t := strings.ToLower(message)
	for _, r := range fallbackRules {
		for _, p := range r.phrases {
			if strings.Contains(t, p) {
				return r.intent
			}
		}
	}
	return OtherChat
}
