package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"healthAgent/internal/service"
)

type handlerSessionRepository struct {
	owners map[string]string
}

func (r *handlerSessionRepository) CreateSession(_ context.Context, userID, sessionID string) (bool, error) {
	if _, exists := r.owners[sessionID]; exists {
		return false, nil
	}
	r.owners[sessionID] = userID
	return true, nil
}

func (r *handlerSessionRepository) OwnsActiveSession(_ context.Context, userID, sessionID string) (bool, error) {
	return r.owners[sessionID] == userID, nil
}

func TestCreateSessionHandlerUsesAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository := &handlerSessionRepository{owners: make(map[string]string)}
	server := &Server{sessions: service.NewSessionService(repository)}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", nil)
	request = request.WithContext(context.WithValue(request.Context(), userIDKey, "usr_owner"))
	ginContext.Request = request

	server.createSessionHandler(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var ownerFound bool
	for sessionID, userID := range repository.owners {
		if !validSessionID(sessionID) {
			t.Fatalf("session ID = %q, want valid backend format", sessionID)
		}
		ownerFound = userID == "usr_owner"
	}
	if !ownerFound {
		t.Fatal("created session is not owned by authenticated user")
	}
}

func TestValidSessionID(t *testing.T) {
	if !validSessionID("session_0123456789abcdef0123456789abcdef") {
		t.Fatal("backend session ID was rejected")
	}
	for _, invalid := range []string{"", "session_bad", "session_01234567-89ab-cdef-0123-456789abcdef"} {
		if validSessionID(invalid) {
			t.Fatalf("invalid session ID %q was accepted", invalid)
		}
	}
}
