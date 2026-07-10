package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	guestUserIDBytes = 16
	guestTokenBytes  = 32
	identityRetries  = 5
)

var ErrUnauthenticated = errors.New("身份凭证无效")

// IdentityRepository 是身份服务需要的最小持久化能力。
type IdentityRepository interface {
	FindActiveGuest(ctx context.Context, tokenHash []byte, now time.Time) (userID string, expiresAt time.Time, found bool, err error)
	CreateGuest(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) (created bool, err error)
}

// GuestIdentity 是一次 Guest 身份恢复或创建的结果。
// Token 只在新建身份时返回，供 HTTP 层写入 HttpOnly Cookie。
type GuestIdentity struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
	Created   bool
}

// IdentityService 编排 Guest 创建、设备身份恢复和后续账号绑定。
type IdentityService struct {
	repository IdentityRepository
	tokenTTL   time.Duration
	random     io.Reader
	now        func() time.Time
}

func NewIdentityService(repository IdentityRepository, tokenTTL time.Duration) *IdentityService {
	return &IdentityService{
		repository: repository,
		tokenTTL:   tokenTTL,
		random:     rand.Reader,
		now:        time.Now,
	}
}

// AuthenticateGuest 只验证已有 Guest 凭证，认证失败时绝不创建新用户。
func (s *IdentityService) AuthenticateGuest(ctx context.Context, rawToken string) (string, error) {
	tokenHash, valid := parseGuestToken(rawToken)
	if !valid {
		return "", ErrUnauthenticated
	}

	userID, _, found, err := s.repository.FindActiveGuest(ctx, tokenHash, s.now().UTC())
	if err != nil {
		return "", fmt.Errorf("认证 Guest 身份失败: %w", err)
	}
	if !found {
		return "", ErrUnauthenticated
	}
	return userID, nil
}

// EnsureGuest 使用有效设备 token 恢复原 Guest；没有有效 token 时原子创建用户和凭证。
func (s *IdentityService) EnsureGuest(ctx context.Context, rawToken string) (GuestIdentity, error) {
	now := s.now().UTC()
	if tokenHash, valid := parseGuestToken(rawToken); valid {
		userID, expiresAt, found, err := s.repository.FindActiveGuest(ctx, tokenHash, now)
		if err != nil {
			return GuestIdentity{}, fmt.Errorf("查询 Guest 身份失败: %w", err)
		}
		if found {
			return GuestIdentity{UserID: userID, ExpiresAt: expiresAt}, nil
		}
	}

	if s.tokenTTL <= 0 {
		return GuestIdentity{}, fmt.Errorf("Guest token 有效期配置无效")
	}

	for range identityRetries {
		userID, err := s.createGuestUserID()
		if err != nil {
			return GuestIdentity{}, fmt.Errorf("生成 Guest user_id 失败: %w", err)
		}
		token, tokenHash, err := s.createGuestToken()
		if err != nil {
			return GuestIdentity{}, fmt.Errorf("生成 Guest token 失败: %w", err)
		}
		expiresAt := now.Add(s.tokenTTL)
		created, err := s.repository.CreateGuest(ctx, userID, tokenHash, expiresAt)
		if err != nil {
			return GuestIdentity{}, fmt.Errorf("创建 Guest 身份失败: %w", err)
		}
		if created {
			return GuestIdentity{
				UserID:    userID,
				Token:     token,
				ExpiresAt: expiresAt,
				Created:   true,
			}, nil
		}
	}

	return GuestIdentity{}, fmt.Errorf("创建 Guest 身份冲突次数过多")
}

func (s *IdentityService) createGuestUserID() (string, error) {
	value := make([]byte, guestUserIDBytes)
	if _, err := io.ReadFull(s.random, value); err != nil {
		return "", err
	}
	return "usr_" + hex.EncodeToString(value), nil
}

func (s *IdentityService) createGuestToken() (string, []byte, error) {
	value := make([]byte, guestTokenBytes)
	if _, err := io.ReadFull(s.random, value); err != nil {
		return "", nil, err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(value)
	tokenHash := sha256.Sum256(value)
	return rawToken, tokenHash[:], nil
}

func parseGuestToken(rawToken string) ([]byte, bool) {
	value, err := base64.RawURLEncoding.DecodeString(rawToken)
	if err != nil || len(value) != guestTokenBytes {
		return nil, false
	}
	tokenHash := sha256.Sum256(value)
	return tokenHash[:], true
}
