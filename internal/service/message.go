package service

import (
	"context"
	"errors"
	"time"
)

const HealthAgentID = "health-agent"

var ErrClientMessageConflict = errors.New("client_message_id 已用于其他消息内容")
var ErrEmptyAssistantMessage = errors.New("assistant 消息不能为空")

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

// AppendAssistantMessageRequest 是写入完整模型回复所需的可信数据。
type AppendAssistantMessageRequest struct {
	UserID    string
	SessionID string
	ParentID  int64
	Content   string
	TraceID   string
}

// AssistantMessage 是已持久化的模型回复。
type AssistantMessage struct {
	ID        int64
	UserID    string
	SessionID string
	Seq       int
	Content   string
	TraceID   string
	CreatedAt time.Time
}

// ConversationMessage 是可进入对话上下文的已完成消息。
type ConversationMessage struct {
	Seq     int
	Role    string
	Content string
}

// MessageRepository 是消息服务当前需要的最小持久化能力。
type MessageRepository interface {
	AppendUserMessage(ctx context.Context, request AppendUserMessageRequest) (AppendUserMessageResult, error)
	AppendAssistantMessage(ctx context.Context, request AppendAssistantMessageRequest) (AssistantMessage, error)
	LoadRecent(ctx context.Context, userID, sessionID string, limit int) ([]ConversationMessage, error)
	// FindAssistantReplyByID 按 turn 保存的结果消息 ID 查已完成的 assistant 回复。
	// 找不到时返回 (zero, false, nil)，不算错误。
	FindAssistantReplyByID(ctx context.Context, userID, sessionID string, messageID int64) (AssistantMessage, bool, error)
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

func (s *MessageService) AppendAssistantMessage(ctx context.Context, request AppendAssistantMessageRequest) (AssistantMessage, error) {
	if request.Content == "" {
		return AssistantMessage{}, ErrEmptyAssistantMessage
	}
	return s.repository.AppendAssistantMessage(ctx, request)
}

// LoadRecent 返回当前用户会话中最近的有效消息，并保持会话内顺序。
func (s *MessageService) LoadRecent(ctx context.Context, userID, sessionID string, limit int) ([]ConversationMessage, error) {
	if limit <= 0 {
		return []ConversationMessage{}, nil
	}
	return s.repository.LoadRecent(ctx, userID, sessionID, limit)
}

// FindReplyForTurn 按 completed turn 持久化的 result_message_id 原样恢复结果。
func (s *MessageService) FindReplyForTurn(ctx context.Context, userID, sessionID string, resultMessageID int64) (AssistantMessage, bool, error) {
	if resultMessageID <= 0 {
		return AssistantMessage{}, false, nil
	}
	return s.repository.FindAssistantReplyByID(ctx, userID, sessionID, resultMessageID)
}
