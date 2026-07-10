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

func (r *fakeSessionRepository) OwnsActiveSession(_ context.Context, userID, sessionID string) (bool, error) {
	return r.owners[sessionID] == userID, nil
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
