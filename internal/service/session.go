package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	sessionIDBytes = 16
	sessionRetries = 5
	// defaultSessionListLimit 是未显式指定时会话列表的条数上限。首版不做搜索和游标分页，
	// 只返回最近活跃的一小批，够前端侧栏用。
	defaultSessionListLimit = 50
)

var ErrSessionNotFound = errors.New("会话不存在")

// SessionListItem 是会话列表里的一项，只暴露列表页需要的字段。
type SessionListItem struct {
	SessionID     string
	Title         string
	Status        string
	MessageCount  int
	LastMessageAt *time.Time
	CreatedAt     time.Time
}

// SessionRepository 是会话服务需要的最小持久化能力。
type SessionRepository interface {
	CreateSession(ctx context.Context, userID, sessionID string) (created bool, err error)
	OwnsSession(ctx context.Context, userID, sessionID string) (bool, error)
	OwnsActiveSession(ctx context.Context, userID, sessionID string) (bool, error)
	// ListSessions 按最近活跃时间倒序返回该用户未删除的会话，limit 限定最大条数。
	ListSessions(ctx context.Context, userID string, limit int) ([]SessionListItem, error)
}

// SessionService 创建会话线程并校验其用户归属。
type SessionService struct {
	repository SessionRepository
	random     io.Reader
}

func NewSessionService(repository SessionRepository) *SessionService {
	return &SessionService{repository: repository, random: rand.Reader}
}

func (s *SessionService) Create(ctx context.Context, userID string) (string, error) {
	for range sessionRetries {
		randomBytes := make([]byte, sessionIDBytes)
		if _, err := io.ReadFull(s.random, randomBytes); err != nil {
			return "", fmt.Errorf("生成 session_id 失败: %w", err)
		}
		sessionID := "session_" + hex.EncodeToString(randomBytes)
		created, err := s.repository.CreateSession(ctx, userID, sessionID)
		if err != nil {
			return "", fmt.Errorf("创建会话失败: %w", err)
		}
		if created {
			return sessionID, nil
		}
	}
	return "", fmt.Errorf("创建会话冲突次数过多")
}

// RequireOwned 校验会话属于当前用户且尚未删除，适用于历史消息等只读操作。
func (s *SessionService) RequireOwned(ctx context.Context, userID, sessionID string) error {
	owned, err := s.repository.OwnsSession(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("校验会话归属失败: %w", err)
	}
	if !owned {
		return ErrSessionNotFound
	}
	return nil
}

// RequireOwnedActive 校验会话属于当前用户、尚未删除且可继续写入。
func (s *SessionService) RequireOwnedActive(ctx context.Context, userID, sessionID string) error {
	owned, err := s.repository.OwnsActiveSession(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("校验会话归属失败: %w", err)
	}
	if !owned {
		return ErrSessionNotFound
	}
	return nil
}

// List 返回当前用户未删除的会话列表，按最近活跃时间倒序。
func (s *SessionService) List(ctx context.Context, userID string) ([]SessionListItem, error) {
	items, err := s.repository.ListSessions(ctx, userID, defaultSessionListLimit)
	if err != nil {
		return nil, fmt.Errorf("查询会话列表失败: %w", err)
	}
	return items, nil
}
