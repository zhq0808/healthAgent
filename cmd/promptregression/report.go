package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeReports(result report, config reportConfig) error {
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("编码 JSON 评测报告失败: %w", err)
	}
	if err := writeReportFile(config.JSONPath, append(encoded, '\n')); err != nil {
		return err
	}
	if err := writeReportFile(config.MarkdownPath, renderMarkdownReport(result)); err != nil {
		return err
	}
	return nil
}

func writeReportFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建报告目录失败: %w", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("写入报告 %s 失败: %w", path, err)
	}
	return nil
}

func renderMarkdownReport(result report) []byte {
	var output bytes.Buffer
	fmt.Fprintf(&output, "# Prompt Evaluation Report\n\n")
	fmt.Fprintf(&output, "- Dataset: `%s` (`%s`)\n", result.DatasetName, result.DatasetVersion)
	fmt.Fprintf(&output, "- Generated: `%s`\n", result.GeneratedAt.Format("2006-01-02 15:04:05Z"))
	fmt.Fprintf(&output, "- Cases: %d\n", result.Summary.Cases)
	fmt.Fprintf(&output, "- Regressions: %d\n", result.Summary.Regressions)
	fmt.Fprintf(&output, "- Improvements: %d\n\n", result.Summary.Improvements)

	output.WriteString("## Runs\n\n| Target | Model | Passed | Failed | Errors | Duration | Tokens |\n|---|---|---:|---:|---:|---:|---:|\n")
	for _, run := range result.Runs {
		fmt.Fprintf(&output, "| %s | %s | %d | %d | %d | %d ms | %d |\n", escapeMarkdown(run.Name), escapeMarkdown(run.Model), run.Passed, run.Failed, run.Errors, run.DurationMillis, run.Tokens.TotalTokens)
	}

	output.WriteString("\n## Comparison\n\n| Case | Task | Category | Baseline | Candidate | Change |\n|---|---|---|---|---|---|\n")
	for _, comparison := range result.Comparisons {
		change := "-"
		if comparison.Regression {
			change = "REGRESSION"
		} else if comparison.Improvement {
			change = "IMPROVEMENT"
		}
		fmt.Fprintf(&output, "| %s | %s | %s | %s | %s | %s |\n", escapeMarkdown(comparison.ID), escapeMarkdown(comparison.TaskType), escapeMarkdown(comparison.Category), comparison.BaselineStatus, comparison.CandidateStatus, change)
	}

	for _, run := range result.Runs {
		fmt.Fprintf(&output, "\n## %s Details\n", escapeMarkdown(run.Name))
		for _, caseResult := range run.Results {
			fmt.Fprintf(&output, "\n### %s\n\n- Status: `%s`\n- Task: `%s`\n- Prompt: `%s`\n- Duration: `%d ms`\n- Tokens: `%d`\n", escapeMarkdown(caseResult.ID), resultStatus(caseResult), escapeMarkdown(caseResult.TaskType), escapeMarkdown(caseResult.PromptVersion), caseResult.DurationMillis, caseResult.Tokens.TotalTokens)
			if len(caseResult.DeterministicScore.Failures) > 0 {
				fmt.Fprintf(&output, "- Failures: %s\n", escapeMarkdown(strings.Join(caseResult.DeterministicScore.Failures, "; ")))
			}
			output.WriteString("\n```text\n")
			output.WriteString(caseResult.RawOutput)
			output.WriteString("\n```\n")
		}
	}
	return output.Bytes()
}

func escapeMarkdown(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
