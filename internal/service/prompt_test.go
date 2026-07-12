package service

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProductChatPromptCoversAINativeRoles(t *testing.T) {
	prompt, err := LoadChatPrompt(
		filepath.Join("..", "..", "prompts", "health_chat_v1.tmpl"),
		"health-chat-v1",
		"不得诊断或开处方",
	)
	if err != nil {
		t.Fatalf("LoadChatPrompt() error = %v", err)
	}

	rendered, err := prompt.Render("目标：控制体重；饮食禁忌：花生过敏")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, required := range []string{
		"忙碌上班族",
		"统一入口",
		"主动管家",
		"解读者",
		"记忆体",
		"今天吃什么",
		"不得伪造执行结果",
		"尚未保存",
		"不能替代医生建议",
		"不得自行缩小或重新解释过敏范围",
		"400 个汉字以内",
		"health-chat-v1",
		"花生过敏",
	} {
		if !strings.Contains(rendered, required) {
			t.Errorf("rendered prompt missing %q", required)
		}
	}
}
