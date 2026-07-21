package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRegressionCasesAreFixedAtTen(t *testing.T) {
	cases, err := loadCases(filepath.Join("..", "..", "prompts", "regression", "cases.json"))
	if err != nil {
		t.Fatalf("loadCases() error = %v", err)
	}
	if len(cases) != 10 {
		t.Fatalf("case count = %d, want 10", len(cases))
	}
	wantCategories := map[string]bool{
		"费曼主动输出":  true,
		"输入不等于掌握": true,
		"事实边界":    true,
		"证据化反馈":   true,
		"诚实不确定性":  true,
		"训练模式":    true,
		"模拟模式":    true,
		"JD 差距分析": true,
		"拒绝编造事实":  true,
		"提示词注入防护": true,
	}
	seen := make(map[string]bool, len(cases))
	for _, testCase := range cases {
		if !wantCategories[testCase.Category] {
			t.Fatalf("unexpected category %q", testCase.Category)
		}
		if seen[testCase.Category] {
			t.Fatalf("duplicate category %q", testCase.Category)
		}
		seen[testCase.Category] = true
	}
}

func TestRegressionCasesRepresentProductPositioning(t *testing.T) {
	cases, err := loadCases(filepath.Join("..", "..", "prompts", "regression", "cases.json"))
	if err != nil {
		t.Fatalf("loadCases() error = %v", err)
	}

	var corpus strings.Builder
	for _, testCase := range cases {
		corpus.WriteString(testCase.Input)
		corpus.WriteString(testCase.Expected)
		corpus.WriteString(testCase.UserFactSummary)
		for _, message := range testCase.History {
			corpus.WriteString(message.Content)
		}
	}
	for _, productAnchor := range []string{"费曼", "主动输出", "生产", "Demo", "模拟面试", "JD", "证据"} {
		if !strings.Contains(corpus.String(), productAnchor) {
			t.Errorf("regression cases missing product anchor %q", productAnchor)
		}
	}
}

func TestEvaluateReportsAllFailedExpectations(t *testing.T) {
	failures := evaluate("可以直接加倍", assertions{
		MustContainAny:  [][]string{{"不要自行", "不能自行"}},
		MustNotContain:  []string{"直接加倍"},
		MustAskQuestion: true,
		MaxRunes:        4,
	})
	if len(failures) != 4 {
		t.Fatalf("failures = %v, want 4 independent failures", failures)
	}
}

func TestEvaluateAcceptsExplicitInformationRequestAsQuestion(t *testing.T) {
	failures := evaluate("请告诉我体检报告中哪项指标异常。", assertions{MustAskQuestion: true})
	if len(failures) != 0 {
		t.Fatalf("failures = %v, want explicit information request to count as a question", failures)
	}
}
