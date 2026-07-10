package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresIdentityRepository 使用共享连接池持久化用户主体和 Guest 设备凭证。
type PostgresIdentityRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresIdentityRepository(pool *pgxpool.Pool) *PostgresIdentityRepository {
	return &PostgresIdentityRepository{pool: pool}
}

// FindActiveGuest 验证凭证、用户类型和账号状态，并更新设备最近使用时间。
func (r *PostgresIdentityRepository) FindActiveGuest(ctx context.Context, tokenHash []byte, now time.Time) (string, time.Time, bool, error) {
	const query = `
		UPDATE guest_credential AS credential
		SET last_used_at = $2
		FROM agent_user AS user_account
		WHERE credential.token_hash = $1
		  AND credential.user_id = user_account.user_id
		  AND credential.revoked_at IS NULL
		  AND credential.expires_at > $2
		  AND user_account.user_type = 0
		  AND user_account.status = 0
		  AND user_account.deleted_at IS NULL
		RETURNING credential.user_id, credential.expires_at`

	var userID string
	var expiresAt time.Time
	err := r.pool.QueryRow(ctx, query, tokenHash, now).Scan(&userID, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, false, nil
	}
	if err != nil {
		return "", time.Time{}, false, fmt.Errorf("查询有效 Guest 凭证失败: %w", err)
	}
	return userID, expiresAt, true, nil
}

// CreateGuest 在同一事务中创建稳定用户主体和设备凭证，不会留下半成品用户。
// 唯一键冲突返回 created=false，由 service 重新生成随机 ID/token 后有限重试。
func (r *PostgresIdentityRepository) CreateGuest(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("开启 Guest 创建事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO agent_user (user_id, user_type, status)
		VALUES ($1, 0, 0)`, userID); err != nil {
		if isUniqueViolation(err) {
			return false, nil
		}
		return false, fmt.Errorf("写入 Guest 用户失败: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO guest_credential (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`, userID, tokenHash, expiresAt); err != nil {
		if isUniqueViolation(err) {
			return false, nil
		}
		return false, fmt.Errorf("写入 Guest 凭证失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("提交 Guest 创建事务失败: %w", err)
	}
	return true, nil
}

func isUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23505"
}
