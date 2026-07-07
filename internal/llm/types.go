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
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// chatCompletionResponse 是 /chat/completions 响应体（仅解析所需字段）。
type chatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// chatCompletionChunk 是 stream=true 时 SSE 每帧 `data:` 的结构（仅解析所需字段）。
// 与非流式的区别：内容在 delta.content 里逐段下发，而非一次性的 message.content。
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
