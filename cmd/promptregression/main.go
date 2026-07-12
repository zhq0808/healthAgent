package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"healthAgent/internal/config"
	"healthAgent/internal/llm"
	"healthAgent/internal/service"
)

type assertions struct {
	MustContainAny  [][]string `json:"must_contain_any"`
	MustNotContain  []string   `json:"must_not_contain"`
	MustAskQuestion bool       `json:"must_ask_question"`
	MaxRunes        int        `json:"max_runes"`
}

type regressionCase struct {
	ID                 string        `json:"id"`
	Category           string        `json:"category"`
	History            []llm.Message `json:"history"`
	Input              string        `json:"input"`
	UserProfileSummary string        `json:"user_profile_summary"`
	Expected           string        `json:"expected"`
	Assertions         assertions    `json:"assertions"`
}

type caseResult struct {
	ID       string   `json:"id"`
	Category string   `json:"category"`
	Expected string   `json:"expected"`
	Passed   bool     `json:"passed"`
	Failures []string `json:"failures,omitempty"`
	Output   string   `json:"output"`
}

type promptRun struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	TemplatePath string       `json:"template_path"`
	Passed       int          `json:"passed"`
	Failed       int          `json:"failed"`
	Results      []caseResult `json:"results"`
}

type report struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Model       string      `json:"model"`
	Temperature float64     `json:"temperature"`
	CasesPath   string      `json:"cases_path"`
	Runs        []promptRun `json:"runs"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config.yaml", "application config path")
	casesPath := flag.String("cases", "prompts/regression/cases.json", "regression cases path")
	baselinePath := flag.String("baseline", "prompts/health_chat_v1.tmpl", "baseline prompt path")
	outputPath := flag.String("out", "docs/0711/prompt-regression-results.json", "JSON report path")
	temperature := flag.Float64("temperature", 0, "fixed temperature used by both runs")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if cfg.DeepSeek.APIKey == "" {
		return errors.New("缺少 DEEPSEEK_API_KEY，未运行回归；不会生成伪造结果")
	}
	cases, err := loadCases(*casesPath)
	if err != nil {
		return err
	}
	if len(cases) != 10 {
		return fmt.Errorf("固定回归用例数量 = %d，want 10", len(cases))
	}

	baselineRaw, err := os.ReadFile(*baselinePath)
	if err != nil {
		return fmt.Errorf("读取 baseline prompt 失败: %w", err)
	}
	baseline, err := service.ParseChatPrompt(string(baselineRaw), "health-chat-v1", cfg.Chat.SafetyBoundary)
	if err != nil {
		return err
	}
	candidate, err := service.LoadChatPrompt(cfg.Chat.PromptPath, cfg.Chat.PromptVersion, cfg.Chat.SafetyBoundary)
	if err != nil {
		return err
	}

	client := llm.NewDeepSeekClient(
		cfg.DeepSeek.APIKey,
		cfg.DeepSeek.BaseURL,
		cfg.DeepSeek.Model,
		*temperature,
		time.Duration(cfg.DeepSeek.TimeoutSeconds)*time.Second,
	)
	runs := make([]promptRun, 0, 2)
	for _, target := range []struct {
		name string
		path string
		data *service.ChatPrompt
	}{
		{name: "baseline", path: *baselinePath, data: baseline},
		{name: "candidate", path: cfg.Chat.PromptPath, data: candidate},
	} {
		result, err := executePromptRun(context.Background(), client, target.name, target.path, target.data, cases)
		if err != nil {
			return err
		}
		runs = append(runs, result)
	}

	result := report{
		GeneratedAt: time.Now().UTC(),
		Model:       cfg.DeepSeek.Model,
		Temperature: *temperature,
		CasesPath:   *casesPath,
		Runs:        runs,
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("编码回归报告失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		return fmt.Errorf("创建报告目录失败: %w", err)
	}
	if err := os.WriteFile(*outputPath, append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("写入回归报告失败: %w", err)
	}
	for _, promptRun := range runs {
		fmt.Printf("%s (%s): %d passed, %d failed\n", promptRun.Name, promptRun.Version, promptRun.Passed, promptRun.Failed)
	}
	fmt.Printf("report: %s\n", *outputPath)
	return nil
}

func loadCases(path string) ([]regressionCase, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取回归用例失败: %w", err)
	}
	var cases []regressionCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		return nil, fmt.Errorf("解析回归用例失败: %w", err)
	}
	for index, testCase := range cases {
		if testCase.ID == "" || testCase.Category == "" || testCase.Input == "" || testCase.Expected == "" {
			return nil, fmt.Errorf("回归用例 #%d 缺少 id/category/input/expected", index+1)
		}
	}
	return cases, nil
}

func executePromptRun(ctx context.Context, client *llm.DeepSeekClient, name, templatePath string, prompt *service.ChatPrompt, cases []regressionCase) (promptRun, error) {
	runResult := promptRun{Name: name, Version: prompt.Version(), TemplatePath: templatePath, Results: make([]caseResult, 0, len(cases))}
	for _, testCase := range cases {
		systemMessage, err := prompt.Render(testCase.UserProfileSummary)
		if err != nil {
			return promptRun{}, err
		}
		messages := make([]llm.Message, 0, len(testCase.History)+2)
		messages = append(messages, llm.Message{Role: "system", Content: systemMessage})
		messages = append(messages, testCase.History...)
		messages = append(messages, llm.Message{Role: "user", Content: testCase.Input})
		var output strings.Builder
		caseCtx, cancel := context.WithTimeout(ctx, client.Timeout())
		err = client.Stream(caseCtx, messages, func(delta string) error {
			output.WriteString(delta)
			return nil
		})
		cancel()
		if err != nil {
			return promptRun{}, fmt.Errorf("%s/%s 调用模型失败: %w", name, testCase.ID, err)
		}
		failures := evaluate(output.String(), testCase.Assertions)
		result := caseResult{
			ID:       testCase.ID,
			Category: testCase.Category,
			Expected: testCase.Expected,
			Passed:   len(failures) == 0,
			Failures: failures,
			Output:   output.String(),
		}
		if result.Passed {
			runResult.Passed++
		} else {
			runResult.Failed++
		}
		runResult.Results = append(runResult.Results, result)
	}
	return runResult, nil
}

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
	for _, forbidden := range checks.MustNotContain {
		if strings.Contains(output, forbidden) {
			failures = append(failures, "包含禁止文本: "+forbidden)
		}
	}
	if checks.MustAskQuestion && !containsQuestion(output) {
		failures = append(failures, "期望追问，但回答中没有问题或信息请求")
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
