package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"healthAgent/internal/service"
)

// PostgresTurnLeaseRepository 使用共享连接池管理会话 turn 占用租约。
//
// Acquire/Release 都只用一个短事务：判断 + 写入在同一次 Begin/Commit 内完成，
// 调用方在事务外拿到结果后再去调 LLM、写 SSE，绝不把事务/行锁带出这两个方法。
type PostgresTurnLeaseRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTurnLeaseRepository(pool *pgxpool.Pool) *PostgresTurnLeaseRepository {
	return &PostgresTurnLeaseRepository{pool: pool}
}

// Acquire 获取（或续期/复用）一个 Session 的 active turn 租约。
//
// 判断顺序：
//  1. 先看这个 client_message_id 是否已有记录——有就是"同一个 turn 在重连/续期"或"历史终态"，
//     不会走到下面的抢占逻辑，也就不会跟自己的旧记录撞唯一约束。
//  2. 没有自己的记录，才去看这个 Session 当前是否被别的 client_message_id 占着：
//     未过期 -> 冲突；已过期 -> 判定占用者已失联，标记 failed 腾出位置。
//  3. 插入新的 active 租约。
func (r *PostgresTurnLeaseRepository) Acquire(ctx context.Context, request service.AcquireTurnLeaseRequest) (service.AcquireTurnLeaseResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return service.AcquireTurnLeaseResult{}, fmt.Errorf("开启获取租约事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()

	own, err := scanTurnLease(tx.QueryRow(ctx, `
		SELECT id, session_id, user_id, client_message_id, status, lease_expires_at, created_at, updated_at
		FROM agent_turn_lease
		WHERE session_id = $1 AND client_message_id = $2
		FOR UPDATE`, request.SessionID, request.ClientMessageID))
	switch {
	case err == nil:
		return r.acquireOwnExisting(ctx, tx, request, own, now)
	case !errors.Is(err, pgx.ErrNoRows):
		return service.AcquireTurnLeaseResult{}, fmt.Errorf("查询自身租约记录失败: %w", err)
	}

	other, err := scanTurnLease(tx.QueryRow(ctx, `
		SELECT id, session_id, user_id, client_message_id, status, lease_expires_at, created_at, updated_at
		FROM agent_turn_lease
		WHERE session_id = $1 AND status = 'active'
		FOR UPDATE`, request.SessionID))
	switch {
	case err == nil:
		if other.LeaseExpiresAt.After(now) {
			return service.AcquireTurnLeaseResult{}, service.ErrTurnLeaseConflict
		}
		// 占用者的租约已过期（大概率进程崩溃/网络异常未释放），判定失败并腾出位置。
		if _, err := tx.Exec(ctx, `
			UPDATE agent_turn_lease SET status = 'failed'
			WHERE id = $1 AND status = 'active'`, other.ID); err != nil {
			return service.AcquireTurnLeaseResult{}, fmt.Errorf("回收过期租约失败: %w", err)
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return service.AcquireTurnLeaseResult{}, fmt.Errorf("查询会话当前租约失败: %w", err)
	}

	inserted, err := scanTurnLease(tx.QueryRow(ctx, `
		INSERT INTO agent_turn_lease (session_id, user_id, client_message_id, status, lease_expires_at)
		VALUES ($1, $2, $3, 'active', $4)
		RETURNING id, session_id, user_id, client_message_id, status, lease_expires_at, created_at, updated_at`,
		request.SessionID, request.UserID, request.ClientMessageID, now.Add(request.LeaseDuration)))
	if err != nil {
		if isForeignKeyViolation(err) {
			return service.AcquireTurnLeaseResult{}, service.ErrSessionNotFound
		}
		if isUniqueViolation(err) {
			// 极小概率的竞态：两个请求几乎同时判定"没有别人占着"，只有一个能插入成功。
			return service.AcquireTurnLeaseResult{}, service.ErrTurnLeaseConflict
		}
		return service.AcquireTurnLeaseResult{}, fmt.Errorf("写入 turn 租约失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return service.AcquireTurnLeaseResult{}, fmt.Errorf("提交获取租约事务失败: %w", err)
	}
	return service.AcquireTurnLeaseResult{Lease: inserted, Acquired: true}, nil
}

// acquireOwnExisting 处理"这个 client_message_id 已有记录"的三种情况：续期、过期后恢复、历史终态。
func (r *PostgresTurnLeaseRepository) acquireOwnExisting(
	ctx context.Context,
	tx pgx.Tx,
	request service.AcquireTurnLeaseRequest,
	existing service.TurnLease,
	now time.Time,
) (service.AcquireTurnLeaseResult, error) {
	if existing.Status != service.TurnLeaseActive {
		// 同一条用户消息之前已经跑到终态，不重复占用/不重复处理，交给结果恢复协议决定怎么答复。
		if err := tx.Commit(ctx); err != nil {
			return service.AcquireTurnLeaseResult{}, fmt.Errorf("提交终态租约查询失败: %w", err)
		}
		return service.AcquireTurnLeaseResult{Lease: existing, Acquired: false}, nil
	}

	// active：无论是否过期，都是同一个 turn 在继续（重连或从崩溃前恢复），直接续期而不新建行。
	renewed, err := scanTurnLease(tx.QueryRow(ctx, `
		UPDATE agent_turn_lease SET lease_expires_at = $1
		WHERE id = $2
		RETURNING id, session_id, user_id, client_message_id, status, lease_expires_at, created_at, updated_at`,
		now.Add(request.LeaseDuration), existing.ID))
	if err != nil {
		return service.AcquireTurnLeaseResult{}, fmt.Errorf("续期 turn 租约失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return service.AcquireTurnLeaseResult{}, fmt.Errorf("提交续期租约事务失败: %w", err)
	}
	return service.AcquireTurnLeaseResult{Lease: renewed, Acquired: true}, nil
}

// Release 把 turn 租约标记为终态。单条 UPDATE 本身就是一次短事务，不需要额外 Begin/Commit。
func (r *PostgresTurnLeaseRepository) Release(ctx context.Context, request service.ReleaseTurnLeaseRequest) error {
	command, err := r.pool.Exec(ctx, `
		UPDATE agent_turn_lease
		SET status = $4
		WHERE session_id = $1 AND user_id = $2 AND client_message_id = $3 AND status = 'active'`,
		request.SessionID, request.UserID, request.ClientMessageID, string(request.Status))
	if err != nil {
		return fmt.Errorf("释放 turn 租约失败: %w", err)
	}
	if command.RowsAffected() == 0 {
		// 不当成硬错误：可能租约已被过期回收逻辑标记为 failed，或本来就不存在，释放本身应当是幂等的。
		return nil
	}
	return nil
}

func isForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23503"
}

func scanTurnLease(row pgx.Row) (service.TurnLease, error) {
	var lease service.TurnLease
	var status string
	err := row.Scan(
		&lease.ID,
		&lease.SessionID,
		&lease.UserID,
		&lease.ClientMessageID,
		&status,
		&lease.LeaseExpiresAt,
		&lease.CreatedAt,
		&lease.UpdatedAt,
	)
	lease.Status = service.TurnLeaseStatus(status)
	return lease, err
}
