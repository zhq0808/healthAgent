// Package prompt 集中管理提示词模板。
//
// 提示词以独立 .md 文件维护（好读、好改、可 diff），通过 go:embed 编译进
// 二进制，部署时零外部依赖。新增提示词：加一个 .md 文件 + 一个导出函数。
package prompt

import (
	_ "embed"
	"strings"
)

//go:embed system.md
var systemTmpl string

// System 返回健康助手的系统人设提示词。
func System() string {
	return strings.TrimSpace(systemTmpl)
}

//go:embed intent.md
var intentTmpl string

// Intent 返回意图识别提示词，userInput 为用户原始输入。
func Intent(userInput string) string {
	return strings.ReplaceAll(intentTmpl, "{user_input}", userInput)
}

//go:embed extract.md
var extractTmpl string

// Extract 返回指标提取提示词，text 为待解析文本（通常是意图识别透传的 extracted_text）。
func Extract(text string) string {
	return strings.ReplaceAll(extractTmpl, "{extracted_text}", text)
}
