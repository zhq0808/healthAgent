// Package prompt 集中管理提示词模板。
//
// 提示词以独立 .txt 文件维护（好读、好改、可 diff），通过 go:embed 编译进
// 二进制，部署时零外部依赖。新增提示词：加一个 files/*.txt + 一个导出函数。
package prompt

import (
	_ "embed"
	"strings"
)

//go:embed files/system.txt
var systemTmpl string

// System 返回通用健康助手的系统人设。
func System() string {
	return strings.TrimSpace(systemTmpl)
}

//go:embed files/router.txt
var routerTmpl string

// Router 返回意图分类提示词，填入用户原始输入。
func Router(userInput string) string {
	return strings.ReplaceAll(routerTmpl, "{user_input}", userInput)
}

//go:embed files/health_extract.txt
var healthExtractTmpl string

// HealthExtract 返回体检指标结构化提取提示词。
func HealthExtract(text string) string {
	return strings.ReplaceAll(healthExtractTmpl, "{extracted_text}", text)
}

//go:embed files/food_record.txt
var foodRecordTmpl string

// FoodRecord 返回饮食记录结构化提示词。
func FoodRecord(text string) string {
	return strings.ReplaceAll(foodRecordTmpl, "{extracted_text}", text)
}

//go:embed files/diet_advice.txt
var dietAdviceTmpl string

// DietAdvice 返回饮食建议提示词模板（用户画像占位符待 S6 填充）。
func DietAdvice() string {
	return dietAdviceTmpl
}
