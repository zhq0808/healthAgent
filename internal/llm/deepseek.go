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

	"healthAgent/internal/prompt"
)

// ErrNotConfigured 表示未配置 API Key，无法调用大模型。调用方应据此走降级逻辑。
var ErrNotConfigured = errors.New("大模型未配置 API Key")

// DeepSeekClient 是 DeepSeek（OpenAI 兼容）对话客户端。
// 并发安全：底层 http.Client 可被多个 goroutine 共用，DeepSeekClient 本身无可变状态。
type DeepSeekClient struct {
	apiKey  string
	baseURL string
	model   string
	timeout time.Duration
	http    *http.Client
}

// NewDeepSeekClient 构造 DeepSeek 客户端。timeout 为单次请求的超时上限，同时作为兜底硬超时挂在 http.Client 上。
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

// Chat 发送单轮健康助手对话，返回模型回复文本。
func (c *DeepSeekClient) Chat(ctx context.Context, message string) (string, error) {
	return c.complete(ctx, []chatMessage{
		{Role: "system", Content: prompt.System()},
		{Role: "user", Content: message},
	})
}

// IntentResult 是意图识别结果，对应 prompt/intent.md 的输出结构。
type IntentResult struct {
	Intent        string  `json:"intent"`
	Confidence    float64 `json:"confidence"`
	ExtractedText string  `json:"extracted_text"`
}

// ClassifyIntent 调用 LLM 对用户输入做意图分类。
// 出错（网络/超时/JSON 不合法/意图为空）时返回 error，调用方应据此走关键词兜底。
func (c *DeepSeekClient) ClassifyIntent(ctx context.Context, userInput string) (IntentResult, error) {
	raw, err := c.complete(ctx, []chatMessage{
		{Role: "user", Content: prompt.Intent(userInput)},
	})
	if err != nil {
		return IntentResult{}, err
	}
	var res IntentResult
	if err := json.Unmarshal([]byte(stripJSONFence(raw)), &res); err != nil {
		return IntentResult{}, fmt.Errorf("解析意图 JSON 失败: %w（原始: %s）", err, raw)
	}
	if res.Intent == "" {
		return IntentResult{}, fmt.Errorf("意图为空（原始: %s）", raw)
	}
	return res, nil
}

// complete 发送一组消息并返回模型回复文本。
//
// ctx 控制取消/超时：传入带 deadline 的 context（例如从请求 context 派生），
// 客户端断开或超时时能主动取消底层 HTTP 请求，不把 goroutine 挂死。
func (c *DeepSeekClient) complete(ctx context.Context, messages []chatMessage) (string, error) {
	if c == nil || c.apiKey == "" {
		return "", ErrNotConfigured
	}

	reqBody := chatCompletionRequest{
		Model:    c.model,
		Messages: messages,
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

// stripJSONFence 去掉模型可能多加的 ```json ``` 代码块包裹，返回纯 JSON。
func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
