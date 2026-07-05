// Package llm 提供大模型调用能力。当前实现为 DeepSeek（OpenAI 兼容 /chat/completions 协议）。
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrNotConfigured 表示未配置 API Key，无法调用大模型。调用方应据此走降级逻辑。
var ErrNotConfigured = errors.New("大模型未配置 API Key")

// systemPrompt 是健康助手的系统设定。S1 先用固定人设；S3 起会拼入用户档案与体检指标。
const systemPrompt = "你是一个专业、亲切的健康管理助手，用简体中文回答用户关于饮食、运动、睡眠的问题。" +
	"回答简洁、可执行，避免长篇大论。你不能替代医生做诊断，遇到严重或紧急症状要提醒用户及时就医。"

// Client 是 DeepSeek（OpenAI 兼容）对话客户端。
// 并发安全：底层 http.Client 可被多个 goroutine 共用，Client 本身无可变状态。
type Client struct {
	apiKey  string
	baseURL string
	model   string
	timeout time.Duration
	http    *http.Client
}

// New 构造 DeepSeek 客户端。timeout 为单次请求的超时上限，同时作为兜底硬超时挂在 http.Client 上。
func New(apiKey, baseURL, model string, timeout time.Duration) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		timeout: timeout,
		http:    &http.Client{Timeout: timeout},
	}
}

// Timeout 返回单次调用的超时上限，供调用方构造带 deadline 的 context。
func (c *Client) Timeout() time.Duration {
	return c.timeout
}

// chatMessage 是 OpenAI 兼容协议的单条消息。
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionRequest 是 /chat/completions 请求体（仅含所需字段）。
type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// chatCompletionResponse 是 /chat/completions 响应体（仅解析所需字段）。
type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// maxResponseBytes 是响应体读取上限（1MB），防止异常大响应耗尽内存。
const maxResponseBytes = 1 << 20

// Chat 发送单轮对话，返回模型回复文本。
//
// ctx 控制取消/超时：调用方应传入带 deadline 的 context（例如从请求 context 派生），
// 这样客户端断开或超时时能主动取消底层 HTTP 请求，不会把 goroutine 挂死。
func (c *Client) Chat(ctx context.Context, message string) (string, error) {
	if c == nil || c.apiKey == "" {
		return "", ErrNotConfigured
	}

	reqBody := chatCompletionRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: message},
		},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用大模型失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("大模型返回状态码 %d: %s", resp.StatusCode, string(body))
	}

	var out chatCompletionResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("大模型返回错误: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("大模型返回空结果")
	}

	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
