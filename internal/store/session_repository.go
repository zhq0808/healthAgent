package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSessionRepository 使用共享连接池持久化会话及其用户归属。
type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

func (r *PostgresSessionRepository) CreateSession(ctx context.Context, userID, sessionID string) (bool, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO agent_memory_session (session_id, user_id)
		VALUES ($1, $2)`, sessionID, userID)
	if err != nil {
		if isUniqueViolation(err) {
			return false, nil
		}
		return false, fmt.Errorf("写入会话失败: %w", err)
	}
	return true, nil
}

func (r *PostgresSessionRepository) OwnsActiveSession(ctx context.Context, userID, sessionID string) (bool, error) {
	var owned bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM agent_memory_session
			WHERE session_id = $1
			  AND user_id = $2
			  AND status = 'active'
			  AND deleted_at IS NULL
		)`, sessionID, userID).Scan(&owned)
	if err != nil {
		return false, fmt.Errorf("查询会话归属失败: %w", err)
	}
	return owned, nil
}
