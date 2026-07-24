package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

func evaluate(output string, checks assertions) []string {
	failures := make([]string, 0)
	for _, alternatives := range checks.MustContainAny {
		matched := false
		for _, alternative := range alternatives {
			if strings.Contains(output, alternative) {
				matched = true
				break
			}
		}
		if !matched {
			failures = append(failures, fmt.Sprintf("缺少任一关键词: %s", strings.Join(alternatives, " | ")))
		}
	}
	for _, required := range checks.MustContainAll {
		if !strings.Contains(output, required) {
			failures = append(failures, "缺少必需文本: "+required)
		}
	}
	for _, forbidden := range checks.MustNotContain {
		if strings.Contains(output, forbidden) {
			failures = append(failures, "包含禁止文本: "+forbidden)
		}
	}
	if checks.MustAskQuestion && !containsQuestion(output) {
		failures = append(failures, "期望追问，但回答中没有问题或信息请求")
	}
	if checks.ValidJSON && !json.Valid([]byte(output)) {
		failures = append(failures, "输出不是有效 JSON")
	}
	if checks.MaxRunes > 0 && utf8.RuneCountInString(output) > checks.MaxRunes {
		failures = append(failures, fmt.Sprintf("回答长度 %d 超过上限 %d", utf8.RuneCountInString(output), checks.MaxRunes))
	}
	return failures
}

func containsQuestion(output string) bool {
	if strings.ContainsAny(output, "?？") {
		return true
	}
	for _, marker := range []string{"请告诉我", "请提供", "请确认", "需要先确认", "想先确认", "麻烦再说"} {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}
