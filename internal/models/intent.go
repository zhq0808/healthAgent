// Package models 定义跨层共享的数据结构与常量。
package models

// 意图类型。值必须与 prompt/files/router.txt 的分类名一致（LLM 按此返回）。
const (
	IntentRecordHealthData = "record_health_data" // 录入身体/体检指标
	IntentRecordFood       = "record_food"        // 记录饮食
	IntentAskDietAdvice    = "ask_diet_advice"    // 询问饮食建议
	IntentOtherChat        = "other_chat"         // 其它闲聊/健康疑问
)

// IntentResult 是意图识别结果，对应 prompt/files/router.txt 的输出结构。
type IntentResult struct {
	Intent        string  `json:"intent"`
	Confidence    float64 `json:"confidence"`
	ExtractedText string  `json:"extracted_text"`
}
