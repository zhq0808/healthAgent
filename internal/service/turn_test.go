package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeTurnLeaseRepository struct {
	acquireResult   AcquireTurnLeaseResult
	acquireErr      error
	releaseErr      error
	completeResult  AssistantMessage
	completeErr     error
	lastAcquireReq  AcquireTurnLeaseRequest
	lastCompleteReq CompleteTurnRequest
	lastReleaseReq  ReleaseTurnLeaseRequest
}

func (r *fakeTurnLeaseRepository) Complete(_ context.Context, request CompleteTurnRequest) (AssistantMessage, error) {
	r.lastCompleteReq = request
	return r.completeResult, r.completeErr
}

func (r *fakeTurnLeaseRepository) Acquire(_ context.Context, request AcquireTurnLeaseRequest) (AcquireTurnLeaseResult, error) {
	r.lastAcquireReq = request
	return r.acquireResult, r.acquireErr
}

func (r *fakeTurnLeaseRepository) Release(_ context.Context, request ReleaseTurnLeaseRequest) error {
	r.lastReleaseReq = request
	return r.releaseErr
}

func TestTurnLeaseServiceAcquireRejectsMissingIdentifiers(t *testing.T) {
	turnLeaseService := NewTurnLeaseService(&fakeTurnLeaseRepository{})

	_, err := turnLeaseService.Acquire(t.Context(), AcquireTurnLeaseRequest{SessionID: "session_a"})
	if err == nil {
		t.Fatal("Acquire() error = nil, want error for missing identifiers")
	}
}

func TestTurnLeaseServiceAcquireAppliesDefaultLeaseDuration(t *testing.T) {
	repository := &fakeTurnLeaseRepository{acquireResult: AcquireTurnLeaseResult{Acquired: true}}
	turnLeaseService := NewTurnLeaseService(repository)

	_, err := turnLeaseService.Acquire(t.Context(), AcquireTurnLeaseRequest{
		UserID:          "usr_owner",
		SessionID:       "session_owner",
		ClientMessageID: "00000000-0000-4000-8000-000000000001",
		Content:         "hello",
	})
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if repository.lastAcquireReq.LeaseDuration != DefaultTurnLeaseDuration {
		t.Fatalf("LeaseDuration = %v, want default %v", repository.lastAcquireReq.LeaseDuration, DefaultTurnLeaseDuration)
	}
}

func TestTurnLeaseServiceAcquireKeepsExplicitLeaseDuration(t *testing.T) {
	repository := &fakeTurnLeaseRepository{acquireResult: AcquireTurnLeaseResult{Acquired: true}}
	turnLeaseService := NewTurnLeaseService(repository)

	explicit := 10 * time.Second
	_, err := turnLeaseService.Acquire(t.Context(), AcquireTurnLeaseRequest{
		UserID:          "usr_owner",
		SessionID:       "session_owner",
		ClientMessageID: "00000000-0000-4000-8000-000000000002",
		Content:         "hello",
		LeaseDuration:   explicit,
	})
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if repository.lastAcquireReq.LeaseDuration != explicit {
		t.Fatalf("LeaseDuration = %v, want %v", repository.lastAcquireReq.LeaseDuration, explicit)
	}
}

func TestTurnLeaseServiceAcquirePropagatesConflict(t *testing.T) {
	repository := &fakeTurnLeaseRepository{acquireErr: ErrTurnLeaseConflict}
	turnLeaseService := NewTurnLeaseService(repository)

	_, err := turnLeaseService.Acquire(t.Context(), AcquireTurnLeaseRequest{
		UserID:          "usr_owner",
		SessionID:       "session_owner",
		ClientMessageID: "00000000-0000-4000-8000-000000000003",
		Content:         "hello",
	})
	if !errors.Is(err, ErrTurnLeaseConflict) {
		t.Fatalf("Acquire() error = %v, want ErrTurnLeaseConflict", err)
	}
}

func TestTurnLeaseServiceReleaseRejectsNonTerminalStatus(t *testing.T) {
	turnLeaseService := NewTurnLeaseService(&fakeTurnLeaseRepository{})

	err := turnLeaseService.Release(t.Context(), ReleaseTurnLeaseRequest{
		UserID:          "usr_owner",
		SessionID:       "session_owner",
		ClientMessageID: "00000000-0000-4000-8000-000000000004",
		AttemptNo:       1,
		Status:          TurnLeaseActive,
	})
	if err == nil {
		t.Fatal("Release() error = nil, want error for non-terminal status")
	}
}

func TestTurnLeaseServiceReleasePassesThroughRepository(t *testing.T) {
	repository := &fakeTurnLeaseRepository{}
	turnLeaseService := NewTurnLeaseService(repository)

	request := ReleaseTurnLeaseRequest{
		UserID:          "usr_owner",
		SessionID:       "session_owner",
		ClientMessageID: "00000000-0000-4000-8000-000000000005",
		AttemptNo:       1,
		Status:          TurnLeaseFailed,
	}
	if err := turnLeaseService.Release(t.Context(), request); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if repository.lastReleaseReq != request {
		t.Fatalf("lastReleaseReq = %+v, want %+v", repository.lastReleaseReq, request)
	}
}

func TestTurnLeaseServiceCompletePassesFencingIdentifiers(t *testing.T) {
	repository := &fakeTurnLeaseRepository{completeResult: AssistantMessage{MessageID: "am-9"}}
	turnLeaseService := NewTurnLeaseService(repository)
	request := CompleteTurnRequest{
		UserID:          "usr_owner",
		SessionID:       "session_owner",
		ClientMessageID: "00000000-0000-4000-8000-000000000006",
		AttemptNo:       2,
		UserMessageID:   "um-7",
		Content:         "reply",
	}

	result, err := turnLeaseService.Complete(t.Context(), request)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if result.MessageID != "am-9" || repository.lastCompleteReq != request {
		t.Fatalf("Complete() result=%+v request=%+v", result, repository.lastCompleteReq)
	}
}
