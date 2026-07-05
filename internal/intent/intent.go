// Package intent 根据用户消息判断对话是否应附带一张结构化卡片。
//
// 设计意图：把「意图识别」收敛到后端，前端只按返回的卡片类型做渲染。
// 当前实现用收紧的短语匹配（够用即可）；后续可无缝替换为 LLM 结构化输出，
// 只要保持 Resolve 的契约不变，前端无需改动。
package intent

import "strings"

// 卡片类型常量。前端按这些值渲染对应卡片。
const (
	CardMeal    = "meal"    // 今日膳食推荐
	CardCheckin = "checkin" // 今日打卡
	CardStats   = "stats"   // 今日数据
)

// rule 是一条意图规则：命中任一短语即返回对应卡片类型。
type rule struct {
	card    string
	phrases []string
}

// rules 按优先级从上到下匹配，命中即返回。
//
// 刻意使用「更完整的短语」而非单字，避免「吃」「饭」「睡眠」这类高频词
// 导致几乎每句话都误弹卡片（见 BUG-002）。
//
// TODO: 意图识别仍需优化。当前短语匹配仍是「够用即可」的临时方案，
// 存在漏召回（换种问法就不弹）与硬编码难维护的问题；
// 后续替换为后端/LLM 结构化意图输出，保持 Resolve 契约不变（见 docs/todo.md）。
var rules = []rule{
	{CardMeal, []string{"吃什么", "吃啥", "吃点什么", "膳食", "食谱", "今天吃", "推荐菜"}},
	{CardCheckin, []string{"打卡"}},
	{CardStats, []string{"今日数据", "今天数据", "身体数据", "数据总览", "今日步数"}},
}

// Resolve 根据用户消息判断是否附带卡片。
// 返回卡片类型（meal/checkin/stats）；ok 为 false 表示纯文本回复、不附卡。
func Resolve(message string) (cardType string, ok bool) {
	t := strings.ToLower(message)
	for _, r := range rules {
		for _, p := range r.phrases {
			if strings.Contains(t, p) {
				return r.card, true
			}
		}
	}
	return "", false
}
