// Package service 编排身份、会话和聊天等业务用例。
package service

import (
	"context"
	"time"

	"healthAgent/internal/llm"
)

const systemPrompt = `你是一个专业、亲切的个人 AI 健康管家，服务于关注体检异常指标与日常饮食的用户。
用简洁、口语化的中文回答，直奔主题，不堆砌套话。
涉及医疗建议时，提醒用户异常情况应及时就医，你不替代专业医生诊断。`

// ChatModel 是聊天服务需要的最小模型能力。
type ChatModel interface {
	Timeout() time.Duration
	Stream(ctx context.Context, messages []llm.Message, onDelta func(delta string) error) error
}

// ChatService 编排聊天上下文和模型调用。
type ChatService struct {
	model ChatModel
}

func NewChatService(model ChatModel) *ChatService {
	return &ChatService{model: model}
}

func (s *ChatService) Timeout() time.Duration {
	return s.model.Timeout()
}

// Stream 组装 system prompt 和服务端读取的可信会话历史，并流式调用模型。
func (s *ChatService) Stream(ctx context.Context, history []ConversationMessage, onDelta func(delta string) error) error {
	messages := make([]llm.Message, 0, len(history)+1)
	messages = append(messages, llm.Message{Role: "system", Content: systemPrompt})
	for _, message := range history {
		messages = append(messages, llm.Message{Role: message.Role, Content: message.Content})
	}
	return s.model.Stream(ctx, messages, onDelta)
}
