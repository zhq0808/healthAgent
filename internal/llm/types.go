package llm

import "errors"

// ErrNotConfigured 表示未配置 API Key，无法调用大模型。调用方应据此走降级逻辑。
var ErrNotConfigured = errors.New("大模型未配置 API Key")

// Message 是 OpenAI 兼容协议的单条对话消息。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionRequest 是 /chat/completions 请求体（仅含所需字段）。
type chatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
}

// TokenUsage 是模型供应商返回的实际 Token 用量。
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Completion 是非流式调用的原始文本和用量，供离线评测等需要完整元数据的场景使用。
type Completion struct {
	Content string     `json:"content"`
	Usage   TokenUsage `json:"usage"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage TokenUsage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// chatCompletionChunk 是 stream=true 时 SSE 每帧 `data:` 的结构（仅解析所需字段）。
type chatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// maxResponseBytes 是响应体读取上限（1MB），防止异常大响应耗尽内存。
const maxResponseBytes = 1 << 20
