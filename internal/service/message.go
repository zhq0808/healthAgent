package service

import (
	"context"
	"errors"
	"time"
)

const HealthAgentID = "health-agent"

var ErrClientMessageConflict = errors.New("client_message_id 已用于其他消息内容")

// AppendUserMessageRequest 是写入一条用户消息所需的可信数据。
type AppendUserMessageRequest struct {
	UserID          string
	SessionID       string
	ClientMessageID string
	Content         string
	TraceID         string
}

// UserMessage 是已持久化的用户消息。
type UserMessage struct {
	ID              int64
	UserID          string
	SessionID       string
	ClientMessageID string
	Seq             int
	Content         string
	TraceID         string
	CreatedAt       time.Time
}

// AppendUserMessageResult 表示消息是首次创建还是幂等命中已有记录。
type AppendUserMessageResult struct {
	Message UserMessage
	Created bool
}

// MessageRepository 是消息服务当前需要的最小持久化能力。
type MessageRepository interface {
	AppendUserMessage(ctx context.Context, request AppendUserMessageRequest) (AppendUserMessageResult, error)
}

// MessageService 编排消息持久化和幂等语义。
type MessageService struct {
	repository MessageRepository
}

func NewMessageService(repository MessageRepository) *MessageService {
	return &MessageService{repository: repository}
}

func (s *MessageService) AppendUserMessage(ctx context.Context, request AppendUserMessageRequest) (AppendUserMessageResult, error) {
	result, err := s.repository.AppendUserMessage(ctx, request)
	if err != nil {
		return AppendUserMessageResult{}, err
	}
	if !result.Created && result.Message.Content != request.Content {
		return AppendUserMessageResult{}, ErrClientMessageConflict
	}
	return result, nil
}
