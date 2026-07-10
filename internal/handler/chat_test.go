package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"healthAgent/internal/llm"
	"healthAgent/internal/service"
)

type handlerMessageRepository struct {
	request service.AppendUserMessageRequest
	result  service.AppendUserMessageResult
	calls   int
}

func (r *handlerMessageRepository) AppendUserMessage(_ context.Context, request service.AppendUserMessageRequest) (service.AppendUserMessageResult, error) {
	r.calls++
	r.request = request
	return r.result, nil
}

type handlerChatModel struct {
	calls int
}

func (m *handlerChatModel) Timeout() time.Duration {
	return time.Second
}

func (m *handlerChatModel) Stream(_ context.Context, _ []llm.Message, onDelta func(string) error) error {
	m.calls++
	return onDelta("reply")
}

func TestChatStreamHandlerPersistsUserMessageBeforeCallingModel(t *testing.T) {
	messageRepository := &handlerMessageRepository{result: service.AppendUserMessageResult{Created: true}}
	chatModel := &handlerChatModel{}
	server := newChatHandlerTestServer(messageRepository, chatModel)

	recorder := performChatRequest(server, `{
		"session_id":"session_0123456789abcdef0123456789abcdef",
		"client_message_id":"550e8400-e29b-41d4-a716-446655440000",
		"message":" hello "
	}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if messageRepository.calls != 1 || chatModel.calls != 1 {
		t.Fatalf("message calls=%d model calls=%d, want 1 and 1", messageRepository.calls, chatModel.calls)
	}
	if messageRepository.request.UserID != "usr_owner" ||
		messageRepository.request.SessionID != "session_0123456789abcdef0123456789abcdef" ||
		messageRepository.request.Content != "hello" {
		t.Fatalf("persist request = %+v", messageRepository.request)
	}
}

func TestChatStreamHandlerDoesNotCallModelForIdempotentRetry(t *testing.T) {
	messageRepository := &handlerMessageRepository{result: service.AppendUserMessageResult{Created: false}}
	chatModel := &handlerChatModel{}
	server := newChatHandlerTestServer(messageRepository, chatModel)

	recorder := performChatRequest(server, `{
		"session_id":"session_0123456789abcdef0123456789abcdef",
		"client_message_id":"550e8400-e29b-41d4-a716-446655440000",
		"message":"hello"
	}`)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	if chatModel.calls != 0 {
		t.Fatalf("model calls = %d, want 0", chatModel.calls)
	}
}

func TestChatStreamHandlerRejectsInvalidClientMessageID(t *testing.T) {
	messageRepository := &handlerMessageRepository{}
	chatModel := &handlerChatModel{}
	server := newChatHandlerTestServer(messageRepository, chatModel)

	recorder := performChatRequest(server, `{
		"session_id":"session_0123456789abcdef0123456789abcdef",
		"client_message_id":"not-a-uuid",
		"message":"hello"
	}`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if messageRepository.calls != 0 || chatModel.calls != 0 {
		t.Fatalf("message calls=%d model calls=%d, want 0 and 0", messageRepository.calls, chatModel.calls)
	}
}

func newChatHandlerTestServer(messageRepository service.MessageRepository, chatModel service.ChatModel) *Server {
	sessionRepository := &handlerSessionRepository{owners: map[string]string{
		"session_0123456789abcdef0123456789abcdef": "usr_owner",
	}}
	return &Server{
		chat:     service.NewChatService(chatModel),
		sessions: service.NewSessionService(sessionRepository),
		messages: service.NewMessageService(messageRepository),
		log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func performChatRequest(server *Server, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/stream", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(context.WithValue(request.Context(), userIDKey, "usr_owner"))
	ginContext.Request = request
	server.chatStreamHandler(ginContext)
	return recorder
}
