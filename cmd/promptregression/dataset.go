package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func loadDataset(path string) (regressionDataset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return regressionDataset{}, fmt.Errorf("读取回归用例失败: %w", err)
	}
	var dataset regressionDataset
	if err := json.Unmarshal(raw, &dataset); err != nil {
		return regressionDataset{}, fmt.Errorf("解析回归用例失败: %w", err)
	}
	if dataset.SchemaVersion != datasetSchemaVersion {
		return regressionDataset{}, fmt.Errorf("不支持的数据集 schema_version %q，want %q", dataset.SchemaVersion, datasetSchemaVersion)
	}
	if strings.TrimSpace(dataset.DatasetVersion) == "" || strings.TrimSpace(dataset.Name) == "" {
		return regressionDataset{}, fmt.Errorf("评测数据集缺少 dataset_version/name")
	}
	if len(dataset.Cases) == 0 {
		return regressionDataset{}, fmt.Errorf("评测数据集没有用例")
	}
	seenIDs := make(map[string]struct{}, len(dataset.Cases))
	for index, testCase := range dataset.Cases {
		if testCase.ID == "" || testCase.TaskType == "" || testCase.Category == "" || testCase.Input == "" || testCase.Expected == "" {
			return regressionDataset{}, fmt.Errorf("回归用例 #%d 缺少 id/task_type/category/input/expected", index+1)
		}
		if _, exists := seenIDs[testCase.ID]; exists {
			return regressionDataset{}, fmt.Errorf("回归用例 id 重复: %s", testCase.ID)
		}
		seenIDs[testCase.ID] = struct{}{}
	}
	return dataset, nil
}

func filterCases(cases []regressionCase, selectedTaskTypes []string) ([]regressionCase, error) {
	if len(selectedTaskTypes) == 0 {
		return append([]regressionCase(nil), cases...), nil
	}
	selected := make(map[string]struct{}, len(selectedTaskTypes))
	for _, taskType := range selectedTaskTypes {
		taskType = strings.TrimSpace(taskType)
		if taskType != "" {
			selected[taskType] = struct{}{}
		}
	}
	available := make(map[string]struct{})
	filtered := make([]regressionCase, 0, len(cases))
	for _, testCase := range cases {
		available[testCase.TaskType] = struct{}{}
		if _, ok := selected[testCase.TaskType]; ok {
			filtered = append(filtered, testCase)
		}
	}
	for taskType := range selected {
		if _, ok := available[taskType]; !ok {
			return nil, fmt.Errorf("数据集不包含任务类型 %q", taskType)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("任务类型过滤后没有评测用例")
	}
	return filtered, nil
}

func taskTypesIn(cases []regressionCase) []string {
	seen := make(map[string]struct{})
	taskTypes := make([]string, 0)
	for _, testCase := range cases {
		if _, ok := seen[testCase.TaskType]; ok {
			continue
		}
		seen[testCase.TaskType] = struct{}{}
		taskTypes = append(taskTypes, testCase.TaskType)
	}
	return taskTypes
}
