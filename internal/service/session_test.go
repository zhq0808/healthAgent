package service

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type fakeSessionRepository struct {
	owners          map[string]string
	createCalls     int
	forcedCollision int
	listResult      []SessionListItem
	listErr         error
	listCalls       int
	lastListUserID  string
	lastListLimit   int
}

func newFakeSessionRepository() *fakeSessionRepository {
	return &fakeSessionRepository{owners: make(map[string]string)}
}

func (r *fakeSessionRepository) CreateSession(_ context.Context, userID, sessionID string) (bool, error) {
	r.createCalls++
	if r.forcedCollision > 0 {
		r.forcedCollision--
		return false, nil
	}
	if _, exists := r.owners[sessionID]; exists {
		return false, nil
	}
	r.owners[sessionID] = userID
	return true, nil
}

func (r *fakeSessionRepository) OwnsSession(_ context.Context, userID, sessionID string) (bool, error) {
	return r.owners[sessionID] == userID, nil
}

func (r *fakeSessionRepository) OwnsActiveSession(_ context.Context, userID, sessionID string) (bool, error) {
	return r.owners[sessionID] == userID, nil
}

func (r *fakeSessionRepository) ListSessions(_ context.Context, userID string, limit int) ([]SessionListItem, error) {
	r.listCalls++
	r.lastListUserID = userID
	r.lastListLimit = limit
	return r.listResult, r.listErr
}

func TestSessionServiceCreatesAndValidatesOwnedSession(t *testing.T) {
	repository := newFakeSessionRepository()
	sessionService := NewSessionService(repository)
	sessionService.random = bytes.NewReader(bytes.Repeat([]byte{0x6e}, 32))

	sessionID, err := sessionService.Create(context.Background(), "usr_a")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(sessionID) != len("session_")+32 {
		t.Fatalf("session ID length = %d, want %d", len(sessionID), len("session_")+32)
	}
	if err := sessionService.RequireOwnedActive(context.Background(), "usr_a", sessionID); err != nil {
		t.Fatalf("RequireOwnedActive() owner error = %v", err)
	}
	if err := sessionService.RequireOwnedActive(context.Background(), "usr_b", sessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("RequireOwnedActive() other user error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionServiceRetriesCollision(t *testing.T) {
	repository := newFakeSessionRepository()
	repository.forcedCollision = 1
	sessionService := NewSessionService(repository)
	sessionService.random = bytes.NewReader(bytes.Repeat([]byte{0x7f}, 32))

	if _, err := sessionService.Create(context.Background(), "usr_a"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repository.createCalls != 2 {
		t.Fatalf("CreateSession() calls = %d, want 2", repository.createCalls)
	}
}

func TestSessionServiceListPassesUserIDAndDefaultLimit(t *testing.T) {
	repository := newFakeSessionRepository()
	repository.listResult = []SessionListItem{{SessionID: "session_a"}, {SessionID: "session_b"}}
	sessionService := NewSessionService(repository)

	items, err := sessionService.List(context.Background(), "usr_a")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() = %+v, want 2 items", items)
	}
	if repository.lastListUserID != "usr_a" {
		t.Fatalf("queried userID = %q, want usr_a", repository.lastListUserID)
	}
	if repository.lastListLimit != defaultSessionListLimit {
		t.Fatalf("queried limit = %d, want default %d", repository.lastListLimit, defaultSessionListLimit)
	}
}

func TestSessionServiceListPropagatesRepositoryError(t *testing.T) {
	repository := newFakeSessionRepository()
	repository.listErr = errors.New("database unavailable")
	sessionService := NewSessionService(repository)

	if _, err := sessionService.List(context.Background(), "usr_a"); err == nil {
		t.Fatal("List() error = nil, want propagated repository error")
	}
}
