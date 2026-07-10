package handler

import (
	"context"
	"errors"
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
	request          service.AppendUserMessageRequest
	result           service.AppendUserMessageResult
	assistantRequest service.AppendAssistantMessageRequest
	assistantErr     error
	calls            int
	assistantCalls   int
	history          []service.ConversationMessage
	loadHistoryCalls int
}

func (r *handlerMessageRepository) AppendUserMessage(_ context.Context, request service.AppendUserMessageRequest) (service.AppendUserMessageResult, error) {
	r.calls++
	r.request = request
	return r.result, nil
}

func (r *handlerMessageRepository) AppendAssistantMessage(_ context.Context, request service.AppendAssistantMessageRequest) (service.AssistantMessage, error) {
	r.assistantCalls++
	r.assistantRequest = request
	return service.AssistantMessage{}, r.assistantErr
}

func (r *handlerMessageRepository) LoadRecent(_ context.Context, _, _ string, _ int) ([]service.ConversationMessage, error) {
	r.loadHistoryCalls++
	return r.history, nil
}

type handlerChatModel struct {
	calls    int
	messages []llm.Message
}

func (m *handlerChatModel) Timeout() time.Duration {
	return time.Second
}

func (m *handlerChatModel) Stream(_ context.Context, messages []llm.Message, onDelta func(string) error) error {
	m.calls++
	m.messages = messages
	return onDelta("reply")
}

func TestChatStreamHandlerPersistsUserMessageBeforeCallingModel(t *testing.T) {
	messageRepository := &handlerMessageRepository{
		result: service.AppendUserMessageResult{
			Message: service.UserMessage{ID: 42},
			Created: true,
		},
		history: []service.ConversationMessage{
			{Seq: 1, Role: "user", Content: "previous question"},
			{Seq: 2, Role: "assistant", Content: "previous answer"},
			{Seq: 3, Role: "user", Content: "hello"},
		},
	}
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
	if messageRepository.calls != 1 || messageRepository.assistantCalls != 1 || chatModel.calls != 1 {
		t.Fatalf("user calls=%d assistant calls=%d model calls=%d, want 1, 1 and 1", messageRepository.calls, messageRepository.assistantCalls, chatModel.calls)
	}
	if messageRepository.loadHistoryCalls != 1 {
		t.Fatalf("history calls=%d, want 1", messageRepository.loadHistoryCalls)
	}
	if len(chatModel.messages) != 4 || chatModel.messages[3] != (llm.Message{Role: "user", Content: "hello"}) {
		t.Fatalf("model messages=%+v, want system plus ordered history", chatModel.messages)
	}
	if messageRepository.request.UserID != "usr_owner" ||
		messageRepository.request.SessionID != "session_0123456789abcdef0123456789abcdef" ||
		messageRepository.request.Content != "hello" {
		t.Fatalf("persist request = %+v", messageRepository.request)
	}
	if messageRepository.assistantRequest.UserID != "usr_owner" ||
		messageRepository.assistantRequest.SessionID != "session_0123456789abcdef0123456789abcdef" ||
		messageRepository.assistantRequest.Content != "reply" {
		t.Fatalf("assistant persist request = %+v", messageRepository.assistantRequest)
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

func TestChatStreamHandlerSendsErrorInsteadOfDoneWhenAssistantPersistenceFails(t *testing.T) {
	messageRepository := &handlerMessageRepository{
		result: service.AppendUserMessageResult{
			Message: service.UserMessage{ID: 42},
			Created: true,
		},
		assistantErr: errors.New("database unavailable"),
	}
	server := newChatHandlerTestServer(messageRepository, &handlerChatModel{})

	recorder := performChatRequest(server, `{
		"session_id":"session_0123456789abcdef0123456789abcdef",
		"client_message_id":"550e8400-e29b-41d4-a716-446655440000",
		"message":"hello"
	}`)

	if !strings.Contains(recorder.Body.String(), "event: error") {
		t.Fatalf("body = %q, want error event", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "event: done") {
		t.Fatalf("body = %q, must not contain done event", recorder.Body.String())
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
