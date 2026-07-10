package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

type guestRecord struct {
	userID    string
	expiresAt time.Time
}

type fakeIdentityRepository struct {
	byHash           map[string]guestRecord
	createCalls      int
	forcedCollisions int
}

func newFakeIdentityRepository() *fakeIdentityRepository {
	return &fakeIdentityRepository{byHash: make(map[string]guestRecord)}
}

func (r *fakeIdentityRepository) FindActiveGuest(_ context.Context, tokenHash []byte, now time.Time) (string, time.Time, bool, error) {
	record, found := r.byHash[hex.EncodeToString(tokenHash)]
	if !found || !record.expiresAt.After(now) {
		return "", time.Time{}, false, nil
	}
	return record.userID, record.expiresAt, true, nil
}

func (r *fakeIdentityRepository) CreateGuest(_ context.Context, userID string, tokenHash []byte, expiresAt time.Time) (bool, error) {
	r.createCalls++
	if r.forcedCollisions > 0 {
		r.forcedCollisions--
		return false, nil
	}
	key := hex.EncodeToString(tokenHash)
	if _, exists := r.byHash[key]; exists {
		return false, nil
	}
	r.byHash[key] = guestRecord{userID: userID, expiresAt: expiresAt}
	return true, nil
}

func TestIdentityServiceEnsureGuestCreatesAndRestoresIdentity(t *testing.T) {
	repository := newFakeIdentityRepository()
	now := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	identityService := NewIdentityService(repository, 24*time.Hour)
	identityService.now = func() time.Time { return now }
	identityService.random = bytes.NewReader(bytes.Repeat([]byte{0x2a}, 96))

	created, err := identityService.EnsureGuest(context.Background(), "")
	if err != nil {
		t.Fatalf("EnsureGuest() create error = %v", err)
	}
	if !created.Created || created.Token == "" {
		t.Fatalf("EnsureGuest() create = %+v, want a new identity with token", created)
	}
	if len(created.UserID) != len("usr_")+32 {
		t.Fatalf("UserID length = %d, want %d", len(created.UserID), len("usr_")+32)
	}

	restored, err := identityService.EnsureGuest(context.Background(), created.Token)
	if err != nil {
		t.Fatalf("EnsureGuest() restore error = %v", err)
	}
	if restored.Created || restored.Token != "" {
		t.Fatalf("EnsureGuest() restore = %+v, want existing identity without a new token", restored)
	}
	if restored.UserID != created.UserID {
		t.Fatalf("restored UserID = %q, want %q", restored.UserID, created.UserID)
	}
	if repository.createCalls != 1 {
		t.Fatalf("CreateGuest() calls = %d, want 1", repository.createCalls)
	}
}

func TestIdentityServiceEnsureGuestReplacesInvalidToken(t *testing.T) {
	repository := newFakeIdentityRepository()
	identityService := NewIdentityService(repository, time.Hour)
	identityService.random = bytes.NewReader(bytes.Repeat([]byte{0x3b}, 64))

	identity, err := identityService.EnsureGuest(context.Background(), "not-a-valid-token")
	if err != nil {
		t.Fatalf("EnsureGuest() error = %v", err)
	}
	if !identity.Created || repository.createCalls != 1 {
		t.Fatalf("EnsureGuest() = %+v, create calls = %d, want replacement identity", identity, repository.createCalls)
	}
}

func TestIdentityServiceEnsureGuestRetriesIdentityCollision(t *testing.T) {
	repository := newFakeIdentityRepository()
	repository.forcedCollisions = 1
	identityService := NewIdentityService(repository, time.Hour)
	identityService.random = bytes.NewReader(bytes.Repeat([]byte{0x4c}, 96))

	identity, err := identityService.EnsureGuest(context.Background(), "")
	if err != nil {
		t.Fatalf("EnsureGuest() error = %v", err)
	}
	if !identity.Created || repository.createCalls != 2 {
		t.Fatalf("EnsureGuest() = %+v, create calls = %d, want success after one collision", identity, repository.createCalls)
	}
}

func TestIdentityServiceAuthenticateGuest(t *testing.T) {
	repository := newFakeIdentityRepository()
	identityService := NewIdentityService(repository, time.Hour)
	identityService.random = bytes.NewReader(bytes.Repeat([]byte{0x5d}, 64))

	created, err := identityService.EnsureGuest(context.Background(), "")
	if err != nil {
		t.Fatalf("EnsureGuest() error = %v", err)
	}

	userID, err := identityService.AuthenticateGuest(context.Background(), created.Token)
	if err != nil {
		t.Fatalf("AuthenticateGuest() error = %v", err)
	}
	if userID != created.UserID {
		t.Fatalf("AuthenticateGuest() userID = %q, want %q", userID, created.UserID)
	}

	if _, err := identityService.AuthenticateGuest(context.Background(), "invalid"); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("AuthenticateGuest() invalid error = %v, want ErrUnauthenticated", err)
	}
}
