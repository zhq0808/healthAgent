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
// 会话归属、历史读取和消息落库将在 repository 接入后继续收敛到这里。
type ChatService struct {
	model ChatModel
}

func NewChatService(model ChatModel) *ChatService {
	return &ChatService{model: model}
}

func (s *ChatService) Timeout() time.Duration {
	return s.model.Timeout()
}

// Stream 组装服务端可信 prompt，并将模型增量传给调用方。
func (s *ChatService) Stream(ctx context.Context, message string, onDelta func(delta string) error) error {
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: message},
	}
	return s.model.Stream(ctx, messages, onDelta)
}
