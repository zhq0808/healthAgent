package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

const (
	sessionIDBytes = 16
	sessionRetries = 5
)

var ErrSessionNotFound = errors.New("会话不存在")

// SessionRepository 是会话服务需要的最小持久化能力。
type SessionRepository interface {
	CreateSession(ctx context.Context, userID, sessionID string) (created bool, err error)
	OwnsActiveSession(ctx context.Context, userID, sessionID string) (bool, error)
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
