package main

import (
	"fmt"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

type promptConfig struct {
	Path      string            `yaml:"path"`
	Version   string            `yaml:"version"`
	Variables map[string]string `yaml:"variables"`
}

type targetConfig struct {
	Name       string                  `yaml:"name"`
	Model      string                  `yaml:"model"`
	Parameters map[string]any          `yaml:"parameters"`
	Prompts    map[string]promptConfig `yaml:"prompts"`
}

type judgeConfig struct {
	Enabled     bool    `yaml:"enabled"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
}

type reportConfig struct {
	JSONPath     string `yaml:"json_path"`
	MarkdownPath string `yaml:"markdown_path"`
}

type evaluationConfig struct {
	DatasetPath string         `yaml:"dataset_path"`
	TaskTypes   []string       `yaml:"task_types"`
	Targets     []targetConfig `yaml:"targets"`
	Judge       judgeConfig    `yaml:"judge"`
	Reports     reportConfig   `yaml:"reports"`
}

func loadEvaluationConfig(path string) (evaluationConfig, error) {
	var config evaluationConfig
	if err := cleanenv.ReadConfig(path, &config); err != nil {
		return evaluationConfig{}, fmt.Errorf("读取评测配置失败: %w", err)
	}
	return config, nil
}

func (config evaluationConfig) validate() error {
	if strings.TrimSpace(config.DatasetPath) == "" {
		return fmt.Errorf("评测配置缺少 dataset_path")
	}
	if strings.TrimSpace(config.Reports.JSONPath) == "" || strings.TrimSpace(config.Reports.MarkdownPath) == "" {
		return fmt.Errorf("评测配置缺少 reports.json_path 或 reports.markdown_path")
	}
	seenTargets := make(map[string]struct{}, len(config.Targets))
	for _, target := range config.Targets {
		if target.Name == "" {
			return fmt.Errorf("评测 target 缺少 name")
		}
		if _, exists := seenTargets[target.Name]; exists {
			return fmt.Errorf("评测 target 名称重复: %s", target.Name)
		}
		seenTargets[target.Name] = struct{}{}
		if _, ok := target.Prompts["default"]; !ok {
			return fmt.Errorf("target %s 缺少 prompts.default", target.Name)
		}
		for taskType, prompt := range target.Prompts {
			if prompt.Path == "" || prompt.Version == "" {
				return fmt.Errorf("target %s 的 prompt %s 缺少 path/version", target.Name, taskType)
			}
		}
	}
	for _, required := range []string{"baseline", "candidate"} {
		if _, ok := seenTargets[required]; !ok {
			return fmt.Errorf("评测配置必须包含 %s target", required)
		}
	}
	return nil
}

func (target targetConfig) promptFor(taskType string) promptConfig {
	if prompt, ok := target.Prompts[taskType]; ok {
		return prompt
	}
	return target.Prompts["default"]
}

func parameterFloat(parameters map[string]any, name string, fallback float64) (float64, error) {
	value, ok := parameters[name]
	if !ok {
		return fallback, nil
	}
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	default:
		return 0, fmt.Errorf("参数 %s 必须是数字", name)
	}
}
