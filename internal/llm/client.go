// Package llm 提供大模型底层调用能力（DeepSeek，OpenAI 兼容 /chat/completions 协议）。
//
// 职责单一：只负责 HTTP 传输——把一组消息发给模型、拿回原始文本。
// 不关心系统人设、提示词拼装、意图解析或降级策略，那些属于上层（prompt/agent）。
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DeepSeekClient 是 DeepSeek（OpenAI 兼容）对话客户端。
//
// 并发安全：底层 http.Client 可被多个 goroutine 共用，DeepSeekClient 自身无可变状态。
type DeepSeekClient struct {
	apiKey  string
	baseURL string
	model   string
	timeout time.Duration
	http    *http.Client
}

// NewDeepSeekClient 构造 DeepSeek 客户端。timeout 为单次请求超时上限，同时作为兜底硬超时挂在 http.Client 上。
func NewDeepSeekClient(apiKey, baseURL, model string, timeout time.Duration) *DeepSeekClient {
	return &DeepSeekClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		timeout: timeout,
		http:    &http.Client{Timeout: timeout},
	}
}

// Timeout 返回单次调用的超时上限，供调用方构造带 deadline 的 context。
func (c *DeepSeekClient) Timeout() time.Duration {
	return c.timeout
}

// Complete 发送一组消息并返回模型回复的原始文本。这是本包对外的唯一调用入口。
//
// ctx 控制取消/超时：传入带 deadline 的 context（例如从请求 context 派生），
// 客户端断开或超时时能主动取消底层 HTTP 请求，避免 goroutine 挂死。
func (c *DeepSeekClient) Complete(ctx context.Context, messages []Message) (string, error) {
	if c == nil || c.apiKey == "" {
		return "", ErrNotConfigured
	}

	body, err := json.Marshal(chatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用大模型失败: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("大模型返回错误: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("大模型返回非 200 状态: %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("大模型返回空结果")
	}

	return parsed.Choices[0].Message.Content, nil
}
