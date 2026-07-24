package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
)

type evaluationPrompt struct {
	version   string
	template  *template.Template
	variables map[string]string
}

type evaluationPromptData struct {
	Version         string
	TaskType        string
	TrustBoundary   string
	UserFactSummary string
	SourceChunks    string
	Variables       map[string]string
}

func loadEvaluationPrompt(config promptConfig) (*evaluationPrompt, error) {
	raw, err := os.ReadFile(config.Path)
	if err != nil {
		return nil, fmt.Errorf("读取评测 prompt %s 失败: %w", config.Path, err)
	}
	parsed, err := template.New("evaluation").Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("解析评测 prompt %s 失败: %w", config.Path, err)
	}
	return &evaluationPrompt{version: config.Version, template: parsed, variables: config.Variables}, nil
}

func (prompt *evaluationPrompt) render(testCase regressionCase, trustBoundary string) (string, error) {
	variables := make(map[string]string, len(prompt.variables)+len(testCase.Context.Variables))
	for key, value := range prompt.variables {
		variables[key] = value
	}
	for key, value := range testCase.Context.Variables {
		variables[key] = value
	}
	userFacts := strings.TrimSpace(testCase.Context.UserFactSummary)
	if userFacts == "" {
		userFacts = "暂无已确认用户事实。"
	}
	sourceChunks := strings.Join(testCase.Context.SourceChunks, "\n")
	if sourceChunks == "" {
		sourceChunks = "暂无来源片段。"
	}
	var rendered bytes.Buffer
	if err := prompt.template.Execute(&rendered, evaluationPromptData{
		Version:         prompt.version,
		TaskType:        testCase.TaskType,
		TrustBoundary:   trustBoundary,
		UserFactSummary: userFacts,
		SourceChunks:    sourceChunks,
		Variables:       variables,
	}); err != nil {
		return "", fmt.Errorf("渲染 %s/%s prompt 失败: %w", testCase.TaskType, testCase.ID, err)
	}
	return rendered.String(), nil
}
