package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"healthAgent/internal/llm"
	"healthAgent/internal/models"
	"healthAgent/internal/prompt"
)

// Analyzer 负责意图分析：优先 LLM，失败降级到关键词兜底。
type Analyzer struct {
	client *llm.DeepSeekClient
	log    *slog.Logger
}

// NewAnalyzer 构建意图分析器。
func NewAnalyzer(client *llm.DeepSeekClient, log *slog.Logger) *Analyzer {
	return &Analyzer{client: client, log: log}
}

// Analyze 返回识别出的意图；两条路径都带 trace_id/user_id 日志，便于按请求链路定位。
func (a *Analyzer) Analyze(ctx context.Context, message, traceID, userID string) models.IntentResult {
	raw, err := a.client.Complete(ctx, []llm.Message{{Role: "user", Content: prompt.Router(message)}})
	if err == nil {
		var res models.IntentResult
		jErr := json.Unmarshal([]byte(stripJSONFence(raw)), &res)
		if jErr == nil && res.Intent != "" {
			a.log.Info("意图识别成功",
				"trace_id", traceID, "user_id", userID,
				"intent", res.Intent, "confidence", res.Confidence)
			return res
		}
		err = fmt.Errorf("解析意图失败: %v（原始: %s）", jErr, raw)
	}

	it := fallback(message)
	a.log.Warn("意图识别降级：LLM 失败，走关键词兜底",
		"trace_id", traceID, "user_id", userID, "intent", it, "error", err)
	return models.IntentResult{Intent: it, ExtractedText: message}
}

// fallbackRule 是一条关键词兜底规则。
type fallbackRule struct {
	intent  string
	phrases []string
}

// 关键词兜底规则：仅在 LLM 分类不可用时使用。
// 用完整短语而非单字，避免「吃」「饭」等高频词误命中（BUG-002）。
var fallbackRules = []fallbackRule{
	{models.IntentRecordHealthData, []string{"血糖", "血压", "血脂", "尿酸", "胆固醇", "甘油三酯", "体重", "体脂", "身高"}},
	{models.IntentRecordFood, []string{"我吃了", "刚吃了", "喝了", "早餐吃", "午餐吃", "晚餐吃"}},
	{models.IntentAskDietAdvice, []string{"吃什么", "吃啥", "吃点什么", "膳食", "食谱", "今天吃", "推荐菜", "能吃"}},
}

// fallback 用关键词匹配兜底分类；兜不中返回 other_chat。
func fallback(message string) string {
	t := strings.ToLower(message)
	for _, r := range fallbackRules {
		for _, p := range r.phrases {
			if strings.Contains(t, p) {
				return r.intent
			}
		}
	}
	return models.IntentOtherChat
}

// stripJSONFence 去掉 LLM 可能包裹的 ```json ... ``` 代码围栏，取出纯 JSON 文本。
func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
