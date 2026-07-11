package store_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"healthAgent/internal/config"
	"healthAgent/internal/service"
	"healthAgent/internal/store"
)

func TestPostgresTurnLeaseRepositoryAcquireAndRelease(t *testing.T) {
	if os.Getenv("HEALTH_AGENT_INTEGRATION_TEST") != "1" {
		t.Skip("set HEALTH_AGENT_INTEGRATION_TEST=1 to run PostgreSQL integration tests")
	}

	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pool, err := store.NewPostgres(cfg.Postgres)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()
	if err := store.RunMigrations(cfg.Postgres, os.DirFS("../../migrations"), "."); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	const (
		userID     = "usr_turn_lease_repo_test"
		otherUser  = "usr_turn_lease_repo_other"
		sessionID  = "session_00000000000000000000000000000090"
		otherOwned = "session_00000000000000000000000000000091"
	)
	ctx := context.Background()
	cleanupTurnLeaseRepositoryTest(t, pool, userID, otherUser)
	defer cleanupTurnLeaseRepositoryTest(t, pool, userID, otherUser)

	if _, err := pool.Exec(ctx, `
		INSERT INTO agent_user (user_id, user_type, status)
		VALUES ($1, 0, 0), ($2, 0, 0)`, userID, otherUser); err != nil {
		t.Fatalf("insert users: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO agent_memory_session (session_id, user_id)
		VALUES ($1, $2), ($3, $4)`, sessionID, userID, otherOwned, otherUser); err != nil {
		t.Fatalf("insert sessions: %v", err)
	}

	turnLeaseService := service.NewTurnLeaseService(store.NewPostgresTurnLeaseRepository(pool))
	firstMessageID := "00000000-0000-4000-8000-000000000090"

	// 1. 首次获取，成功持有 active 租约。
	first, err := turnLeaseService.Acquire(ctx, service.AcquireTurnLeaseRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: firstMessageID,
		LeaseDuration:   time.Minute,
	})
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if !first.Acquired || first.Lease.Status != service.TurnLeaseActive {
		t.Fatalf("first acquire result = %+v, want acquired active", first)
	}

	// 2. 同一 client_message_id 重试（模拟断线重连），复用同一行并续期，不新建记录。
	renewed, err := turnLeaseService.Acquire(ctx, service.AcquireTurnLeaseRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: firstMessageID,
		LeaseDuration:   time.Minute,
	})
	if err != nil {
		t.Fatalf("renew acquire: %v", err)
	}
	if !renewed.Acquired || renewed.Lease.ID != first.Lease.ID {
		t.Fatalf("renew acquire result = %+v, want acquired same id %d", renewed, first.Lease.ID)
	}
	if !renewed.Lease.LeaseExpiresAt.After(first.Lease.LeaseExpiresAt) {
		t.Fatalf("renewed lease_expires_at = %v, want after %v", renewed.Lease.LeaseExpiresAt, first.Lease.LeaseExpiresAt)
	}

	// 3. 别的 client_message_id 在租约未过期时来抢，必须冲突。
	secondMessageID := "00000000-0000-4000-8000-000000000091"
	_, err = turnLeaseService.Acquire(ctx, service.AcquireTurnLeaseRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: secondMessageID,
		LeaseDuration:   time.Minute,
	})
	if !errors.Is(err, service.ErrTurnLeaseConflict) {
		t.Fatalf("conflicting acquire error = %v, want ErrTurnLeaseConflict", err)
	}

	// 4. 释放后，别的 client_message_id 才能获取新的 active 租约。
	if err := turnLeaseService.Release(ctx, service.ReleaseTurnLeaseRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: firstMessageID,
		Status:          service.TurnLeaseCompleted,
	}); err != nil {
		t.Fatalf("release first lease: %v", err)
	}
	second, err := turnLeaseService.Acquire(ctx, service.AcquireTurnLeaseRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: secondMessageID,
		LeaseDuration:   time.Minute,
	})
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	if !second.Acquired || second.Lease.ID == first.Lease.ID {
		t.Fatalf("acquire after release result = %+v, want a new active lease", second)
	}

	// 5. 已终态的 client_message_id 再次获取：不重复占用，直接把历史终态原样返回。
	terminalHit, err := turnLeaseService.Acquire(ctx, service.AcquireTurnLeaseRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: firstMessageID,
		LeaseDuration:   time.Minute,
	})
	if err != nil {
		t.Fatalf("acquire terminal client_message_id: %v", err)
	}
	if terminalHit.Acquired || terminalHit.Lease.Status != service.TurnLeaseCompleted {
		t.Fatalf("terminal acquire result = %+v, want not acquired + completed", terminalHit)
	}

	// 释放掉步骤 4 留下的 active 租约，让 Session 回到空闲，才能测下面的过期抢占场景。
	if err := turnLeaseService.Release(ctx, service.ReleaseTurnLeaseRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: secondMessageID,
		Status:          service.TurnLeaseCompleted,
	}); err != nil {
		t.Fatalf("release second lease: %v", err)
	}

	// 6. 已过期但仍标记 active 的旧占用者应被判定失败并腾出位置，新 client_message_id 可以拿到租约。
	expiringMessageID := "00000000-0000-4000-8000-000000000092"
	expiring, err := turnLeaseService.Acquire(ctx, service.AcquireTurnLeaseRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: expiringMessageID,
		LeaseDuration:   time.Millisecond,
	})
	if err != nil || !expiring.Acquired {
		t.Fatalf("acquire soon-to-expire lease: result=%+v err=%v", expiring, err)
	}
	time.Sleep(20 * time.Millisecond)

	reclaimMessageID := "00000000-0000-4000-8000-000000000093"
	reclaimed, err := turnLeaseService.Acquire(ctx, service.AcquireTurnLeaseRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: reclaimMessageID,
		LeaseDuration:   time.Minute,
	})
	if err != nil {
		t.Fatalf("reclaim expired lease: %v", err)
	}
	if !reclaimed.Acquired || reclaimed.Lease.ClientMessageID != reclaimMessageID {
		t.Fatalf("reclaim result = %+v, want acquired for %s", reclaimed, reclaimMessageID)
	}
	var expiredStatus string
	if err := pool.QueryRow(ctx, `
		SELECT status FROM agent_turn_lease WHERE session_id = $1 AND client_message_id = $2`,
		sessionID, expiringMessageID).Scan(&expiredStatus); err != nil {
		t.Fatalf("query reclaimed-away lease status: %v", err)
	}
	if expiredStatus != string(service.TurnLeaseFailed) {
		t.Fatalf("reclaimed-away lease status = %s, want failed", expiredStatus)
	}

	// 释放掉步骤 6 留下的 active 租约，确保下面的 FK 校验测的是"用户不归属该会话"，
	// 而不是先撞上"会话已被占用"的冲突分支。
	if err := turnLeaseService.Release(ctx, service.ReleaseTurnLeaseRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: reclaimMessageID,
		Status:          service.TurnLeaseCompleted,
	}); err != nil {
		t.Fatalf("release reclaimed lease: %v", err)
	}

	// 7. session/user 不匹配（无归属关系）必须返回 ErrSessionNotFound，不能把租约挂到别人会话上。
	_, err = turnLeaseService.Acquire(ctx, service.AcquireTurnLeaseRequest{
		UserID:          otherUser,
		SessionID:       sessionID,
		ClientMessageID: "00000000-0000-4000-8000-000000000094",
		LeaseDuration:   time.Minute,
	})
	if !errors.Is(err, service.ErrSessionNotFound) {
		t.Fatalf("mismatched owner acquire error = %v, want ErrSessionNotFound", err)
	}
}

// TestPostgresTurnLeaseRepositoryAcquireIsSerializedAcrossInstances 用两个独立的连接池
// （模拟两个进程/两个 repository 实例）并发抢同一个 Session 的 turn，验证互斥由数据库短事务
// 保证，而不是靠 Go 进程内的内存锁。
func TestPostgresTurnLeaseRepositoryAcquireIsSerializedAcrossInstances(t *testing.T) {
	if os.Getenv("HEALTH_AGENT_INTEGRATION_TEST") != "1" {
		t.Skip("set HEALTH_AGENT_INTEGRATION_TEST=1 to run PostgreSQL integration tests")
	}

	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	poolA, err := store.NewPostgres(cfg.Postgres)
	if err != nil {
		t.Fatalf("connect postgres pool A: %v", err)
	}
	defer poolA.Close()
	if err := store.RunMigrations(cfg.Postgres, os.DirFS("../../migrations"), "."); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	poolB, err := store.NewPostgres(cfg.Postgres)
	if err != nil {
		t.Fatalf("connect postgres pool B: %v", err)
	}
	defer poolB.Close()

	const (
		userID    = "usr_turn_lease_cross_instance_test"
		sessionID = "session_00000000000000000000000000000095"
	)
	ctx := context.Background()
	cleanupTurnLeaseRepositoryTest(t, poolA, userID)
	defer cleanupTurnLeaseRepositoryTest(t, poolA, userID)

	if _, err := poolA.Exec(ctx, `
		INSERT INTO agent_user (user_id, user_type, status)
		VALUES ($1, 0, 0)`, userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := poolA.Exec(ctx, `
		INSERT INTO agent_memory_session (session_id, user_id)
		VALUES ($1, $2)`, sessionID, userID); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	serviceA := service.NewTurnLeaseService(store.NewPostgresTurnLeaseRepository(poolA))
	serviceB := service.NewTurnLeaseService(store.NewPostgresTurnLeaseRepository(poolB))

	const attempts = 8
	type outcome struct {
		acquired bool
		err      error
	}
	results := make(chan outcome, attempts)
	for index := 0; index < attempts; index++ {
		turnService, clientMessageID := serviceA, fmt.Sprintf("00000000-0000-4000-8000-1%011d", index)
		if index%2 == 1 {
			turnService = serviceB
		}
		go func(turnService *service.TurnLeaseService, clientMessageID string) {
			result, acquireErr := turnService.Acquire(ctx, service.AcquireTurnLeaseRequest{
				UserID:          userID,
				SessionID:       sessionID,
				ClientMessageID: clientMessageID,
				LeaseDuration:   time.Minute,
			})
			results <- outcome{acquired: result.Acquired, err: acquireErr}
		}(turnService, clientMessageID)
	}

	successes := 0
	conflicts := 0
	for index := 0; index < attempts; index++ {
		result := <-results
		switch {
		case result.err == nil && result.acquired:
			successes++
		case errors.Is(result.err, service.ErrTurnLeaseConflict):
			conflicts++
		default:
			t.Fatalf("unexpected acquire outcome: %+v", result)
		}
	}
	if successes != 1 || conflicts != attempts-1 {
		t.Fatalf("successes=%d conflicts=%d, want exactly 1 success and %d conflicts", successes, conflicts, attempts-1)
	}

	var activeCount int
	if err := poolA.QueryRow(ctx, `
		SELECT COUNT(*) FROM agent_turn_lease WHERE session_id = $1 AND status = 'active'`,
		sessionID).Scan(&activeCount); err != nil {
		t.Fatalf("count active leases: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("active lease count = %d, want 1", activeCount)
	}
}

func cleanupTurnLeaseRepositoryTest(t *testing.T, pool *pgxpool.Pool, userIDs ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, userID := range userIDs {
		if _, err := pool.Exec(ctx, `DELETE FROM agent_turn_lease WHERE user_id = $1`, userID); err != nil {
			t.Fatalf("cleanup turn leases: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM agent_memory_session WHERE user_id = $1`, userID); err != nil {
			t.Fatalf("cleanup sessions: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM agent_user WHERE user_id = $1`, userID); err != nil {
			t.Fatalf("cleanup user: %v", err)
		}
	}
}
