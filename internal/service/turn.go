package service

import (
	"context"
	"errors"
	"time"
)

// ErrTurnLeaseConflict 表示该 Session 已有另一个未过期的 active turn，本次获取必须被拒绝（对应未来的 409）。
var ErrTurnLeaseConflict = errors.New("session 已有进行中的 turn")

// TurnLeaseStatus 是 turn 租约的状态取值，对应数据库 agent_turn_lease.status。
type TurnLeaseStatus string

const (
	TurnLeaseActive    TurnLeaseStatus = "active"
	TurnLeaseCompleted TurnLeaseStatus = "completed"
	TurnLeaseFailed    TurnLeaseStatus = "failed"
)

// DefaultTurnLeaseDuration 是未显式指定租期时的默认租约时长。
// 需要覆盖"调用 LLM + 写 SSE"的正常耗时，但不能长到进程崩溃后卡住 Session 太久。
const DefaultTurnLeaseDuration = 45 * time.Second

// TurnLease 是当前持有（或历史命中）的一条 turn 租约记录。
type TurnLease struct {
	ID              int64
	SessionID       string
	UserID          string
	ClientMessageID string
	Status          TurnLeaseStatus
	LeaseExpiresAt  time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AcquireTurnLeaseRequest 是获取（或续期/复用）一个 turn 租约所需的信息。
type AcquireTurnLeaseRequest struct {
	UserID          string
	SessionID       string
	ClientMessageID string
	LeaseDuration   time.Duration
}

// AcquireTurnLeaseResult 描述本次获取的结果。
//
// Acquired=true 表示调用方现在持有这条 active 租约（新建、过期抢占重建，或同一 client_message_id 续期/复用）。
// Acquired=false 且 err=nil 表示命中了同一 client_message_id 的历史终态记录（completed/failed）——
// 不是错误，而是"这条用户消息之前已经处理完了"，具体怎么恢复结果留给后续的结果恢复协议处理。
type AcquireTurnLeaseResult struct {
	Lease    TurnLease
	Acquired bool
}

// ReleaseTurnLeaseRequest 是释放（结束）一个 turn 租约所需的信息。
type ReleaseTurnLeaseRequest struct {
	UserID          string
	SessionID       string
	ClientMessageID string
	Status          TurnLeaseStatus // 只能是 completed 或 failed，表示这个 turn 走到了终态
}

// TurnLeaseRepository 是 turn 租约需要的最小持久化能力；具体的短事务和并发裁决在实现里完成。
type TurnLeaseRepository interface {
	Acquire(ctx context.Context, request AcquireTurnLeaseRequest) (AcquireTurnLeaseResult, error)
	Release(ctx context.Context, request ReleaseTurnLeaseRequest) error
}

// TurnLeaseService 编排 turn 租约获取/释放的入参校验，具体并发语义交给 repository。
type TurnLeaseService struct {
	repository TurnLeaseRepository
}

func NewTurnLeaseService(repository TurnLeaseRepository) *TurnLeaseService {
	return &TurnLeaseService{repository: repository}
}

// Acquire 获取（或续期/复用）Session 的 active turn 租约。
func (s *TurnLeaseService) Acquire(ctx context.Context, request AcquireTurnLeaseRequest) (AcquireTurnLeaseResult, error) {
	if request.UserID == "" || request.SessionID == "" || request.ClientMessageID == "" {
		return AcquireTurnLeaseResult{}, errors.New("获取 turn 租约缺少必要标识")
	}
	if request.LeaseDuration <= 0 {
		request.LeaseDuration = DefaultTurnLeaseDuration
	}
	return s.repository.Acquire(ctx, request)
}

// Release 把 turn 租约标记为终态（completed/failed），归还占用。
func (s *TurnLeaseService) Release(ctx context.Context, request ReleaseTurnLeaseRequest) error {
	if request.UserID == "" || request.SessionID == "" || request.ClientMessageID == "" {
		return errors.New("释放 turn 租约缺少必要标识")
	}
	if request.Status != TurnLeaseCompleted && request.Status != TurnLeaseFailed {
		return errors.New("释放 turn 租约的终态只能是 completed 或 failed")
	}
	return s.repository.Release(ctx, request)
}
