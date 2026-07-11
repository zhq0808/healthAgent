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
	reply            service.AssistantMessage
	replyFound       bool
	replyErr         error
	findReplyCalls   int
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

func (r *handlerMessageRepository) FindAssistantReplyByID(_ context.Context, _, _ string, _ int64) (service.AssistantMessage, bool, error) {
	r.findReplyCalls++
	return r.reply, r.replyFound, r.replyErr
}

type handlerTurnLeaseRepository struct {
	acquireResult service.AcquireTurnLeaseResult
	acquireErr    error
	releaseErr    error
	completeErr   error
	acquireCalls  int
	completeCalls int
	releaseCalls  int
	lastComplete  service.CompleteTurnRequest
	lastRelease   service.ReleaseTurnLeaseRequest
}

func (r *handlerTurnLeaseRepository) Acquire(_ context.Context, _ service.AcquireTurnLeaseRequest) (service.AcquireTurnLeaseResult, error) {
	r.acquireCalls++
	if r.acquireResult.Acquired {
		if r.acquireResult.Lease.AttemptNo == 0 {
			r.acquireResult.Lease.AttemptNo = 1
		}
		if r.acquireResult.UserMessage.ID == 0 {
			r.acquireResult.UserMessage.ID = 42
		}
	}
	return r.acquireResult, r.acquireErr
}

func (r *handlerTurnLeaseRepository) Complete(_ context.Context, request service.CompleteTurnRequest) (service.AssistantMessage, error) {
	r.completeCalls++
	r.lastComplete = request
	return service.AssistantMessage{ID: 99, Content: request.Content}, r.completeErr
}

func (r *handlerTurnLeaseRepository) Release(_ context.Context, request service.ReleaseTurnLeaseRequest) error {
	r.releaseCalls++
	r.lastRelease = request
	return r.releaseErr
}

type handlerChatModel struct {
	calls     int
	messages  []llm.Message
	streamErr error
}

func (m *handlerChatModel) Timeout() time.Duration {
	return time.Second
}

func (m *handlerChatModel) Stream(_ context.Context, messages []llm.Message, onDelta func(string) error) error {
	m.calls++
	m.messages = messages
	if m.streamErr != nil {
		return m.streamErr
	}
	return onDelta("reply")
}

func TestChatStreamHandlerBeginsAndCompletesTurnAroundModelCall(t *testing.T) {
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
	turnLeaseRepository := &handlerTurnLeaseRepository{acquireResult: service.AcquireTurnLeaseResult{Acquired: true}}
	server := newChatHandlerTestServerWithLease(messageRepository, chatModel, turnLeaseRepository)

	recorder := performChatRequest(server, `{
		"session_id":"session_0123456789abcdef0123456789abcdef",
		"client_message_id":"550e8400-e29b-41d4-a716-446655440000",
		"message":" hello "
	}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if messageRepository.calls != 0 || messageRepository.assistantCalls != 0 || chatModel.calls != 1 {
		t.Fatalf("direct user calls=%d direct assistant calls=%d model calls=%d, want 0, 0 and 1", messageRepository.calls, messageRepository.assistantCalls, chatModel.calls)
	}
	if messageRepository.loadHistoryCalls != 1 {
		t.Fatalf("history calls=%d, want 1", messageRepository.loadHistoryCalls)
	}
	if len(chatModel.messages) != 4 || chatModel.messages[3] != (llm.Message{Role: "user", Content: "hello"}) {
		t.Fatalf("model messages=%+v, want system plus ordered history", chatModel.messages)
	}
	if turnLeaseRepository.completeCalls != 1 || turnLeaseRepository.lastComplete.UserMessageID != 42 ||
		turnLeaseRepository.lastComplete.Content != "reply" {
		t.Fatalf("complete calls=%d request=%+v", turnLeaseRepository.completeCalls, turnLeaseRepository.lastComplete)
	}
}

// 结果恢复协议：同一条 client_message_id 重复提交，且对应的 turn 之前已经 completed，
// 必须原样回放当年落库的 assistant 回复，绝不重新调用模型。
func TestChatStreamHandlerReplaysCompletedReplyForIdempotentRetryWithoutCallingModel(t *testing.T) {
	messageRepository := &handlerMessageRepository{
		result: service.AppendUserMessageResult{
			Message: service.UserMessage{ID: 42, Seq: 3, Content: "hello"},
			Created: false,
		},
		reply:      service.AssistantMessage{Content: "previous reply"},
		replyFound: true,
	}
	chatModel := &handlerChatModel{}
	turnLeaseRepository := &handlerTurnLeaseRepository{
		acquireResult: service.AcquireTurnLeaseResult{
			Lease:    service.TurnLease{Status: service.TurnLeaseCompleted, ResultMessageID: 99},
			Acquired: false,
		},
	}
	server := newChatHandlerTestServerWithLease(messageRepository, chatModel, turnLeaseRepository)

	recorder := performChatRequest(server, `{
		"session_id":"session_0123456789abcdef0123456789abcdef",
		"client_message_id":"550e8400-e29b-41d4-a716-446655440000",
		"message":"hello"
	}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if chatModel.calls != 0 {
		t.Fatalf("model calls = %d, want 0: a completed turn must not call the model again", chatModel.calls)
	}
	if messageRepository.assistantCalls != 0 {
		t.Fatalf("assistant calls = %d, want 0: replay must not persist a new assistant message", messageRepository.assistantCalls)
	}
	if messageRepository.findReplyCalls != 1 {
		t.Fatalf("find reply calls = %d, want 1", messageRepository.findReplyCalls)
	}
	if !strings.Contains(recorder.Body.String(), "previous reply") {
		t.Fatalf("body = %q, want the previously persisted reply replayed verbatim", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "event: done") {
		t.Fatalf("body = %q, want done event", recorder.Body.String())
	}
	if turnLeaseRepository.releaseCalls != 0 {
		t.Fatalf("release calls = %d, want 0: replay never acquired a lease, so there is nothing to release", turnLeaseRepository.releaseCalls)
	}
}

// 数据不一致兜底：租约说已经 completed，却查不到对应的 assistant 回复，必须报服务不可用，
// 不能凭空编造内容，也不能借机重新调用模型。
func TestChatStreamHandlerFailsWhenCompletedReplyIsMissing(t *testing.T) {
	messageRepository := &handlerMessageRepository{
		result: service.AppendUserMessageResult{
			Message: service.UserMessage{ID: 42, Seq: 3, Content: "hello"},
			Created: false,
		},
		replyFound: false,
	}
	chatModel := &handlerChatModel{}
	turnLeaseRepository := &handlerTurnLeaseRepository{
		acquireResult: service.AcquireTurnLeaseResult{
			Lease:    service.TurnLease{Status: service.TurnLeaseCompleted, ResultMessageID: 99},
			Acquired: false,
		},
	}
	server := newChatHandlerTestServerWithLease(messageRepository, chatModel, turnLeaseRepository)

	recorder := performChatRequest(server, `{
		"session_id":"session_0123456789abcdef0123456789abcdef",
		"client_message_id":"550e8400-e29b-41d4-a716-446655440000",
		"message":"hello"
	}`)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
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
	}
	turnLeaseRepository := &handlerTurnLeaseRepository{
		acquireResult: service.AcquireTurnLeaseResult{Acquired: true},
		completeErr:   errors.New("database unavailable"),
	}
	server := newChatHandlerTestServerWithLease(messageRepository, &handlerChatModel{}, turnLeaseRepository)

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
	// assistant 写入失败属于本轮没跑到终态，释放时必须标记为 failed 而不是 completed，
	// 否则下一个重试会被误当成“已处理完成”跳过。
	if turnLeaseRepository.releaseCalls != 1 || turnLeaseRepository.lastRelease.Status != service.TurnLeaseFailed {
		t.Fatalf("release calls=%d status=%v, want 1 release with failed status", turnLeaseRepository.releaseCalls, turnLeaseRepository.lastRelease.Status)
	}
}

func TestChatStreamHandlerReturnsConflictWhenTurnLeaseHeldBySomeoneElse(t *testing.T) {
	messageRepository := &handlerMessageRepository{
		result: service.AppendUserMessageResult{
			Message: service.UserMessage{ID: 42},
			Created: true,
		},
	}
	chatModel := &handlerChatModel{}
	turnLeaseRepository := &handlerTurnLeaseRepository{acquireErr: service.ErrTurnLeaseConflict}
	server := newChatHandlerTestServerWithLease(messageRepository, chatModel, turnLeaseRepository)

	recorder := performChatRequest(server, `{
		"session_id":"session_0123456789abcdef0123456789abcdef",
		"client_message_id":"550e8400-e29b-41d4-a716-446655440000",
		"message":"hello"
	}`)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}
	if chatModel.calls != 0 {
		t.Fatalf("model calls = %d, want 0", chatModel.calls)
	}
	if messageRepository.calls != 0 {
		t.Fatalf("user message calls = %d, want 0: conflicting turn transaction must roll back its user message", messageRepository.calls)
	}
	if turnLeaseRepository.releaseCalls != 0 {
		t.Fatalf("release calls = %d, want 0 because the lease was never acquired", turnLeaseRepository.releaseCalls)
	}
}

func TestChatStreamHandlerCompletesTurnAtomicallyOnNormalSuccess(t *testing.T) {
	messageRepository := &handlerMessageRepository{
		result: service.AppendUserMessageResult{
			Message: service.UserMessage{ID: 42},
			Created: true,
		},
	}
	turnLeaseRepository := &handlerTurnLeaseRepository{acquireResult: service.AcquireTurnLeaseResult{Acquired: true}}
	server := newChatHandlerTestServerWithLease(messageRepository, &handlerChatModel{}, turnLeaseRepository)

	recorder := performChatRequest(server, `{
		"session_id":"session_0123456789abcdef0123456789abcdef",
		"client_message_id":"550e8400-e29b-41d4-a716-446655440000",
		"message":"hello"
	}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if turnLeaseRepository.acquireCalls != 1 || turnLeaseRepository.completeCalls != 1 || turnLeaseRepository.releaseCalls != 0 {
		t.Fatalf("acquire=%d complete=%d release=%d, want 1, 1 and 0", turnLeaseRepository.acquireCalls, turnLeaseRepository.completeCalls, turnLeaseRepository.releaseCalls)
	}
}

// LLM 调用失败、请求超时、客户端断开在代码里走的是同一条错误分支（Stream 返回一个非 ErrNotConfigured
// 的 error），这里用“LLM 直接报错”代表这一类场景，验证租约会被释放为 failed 且不会错误地继给 assistant 落库。
func TestChatStreamHandlerReleasesLeaseAsFailedWhenModelStreamFails(t *testing.T) {
	messageRepository := &handlerMessageRepository{
		result: service.AppendUserMessageResult{
			Message: service.UserMessage{ID: 42},
			Created: true,
		},
	}
	chatModel := &handlerChatModel{streamErr: errors.New("upstream timeout")}
	turnLeaseRepository := &handlerTurnLeaseRepository{acquireResult: service.AcquireTurnLeaseResult{Acquired: true}}
	server := newChatHandlerTestServerWithLease(messageRepository, chatModel, turnLeaseRepository)

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
	if messageRepository.assistantCalls != 0 {
		t.Fatalf("assistant calls = %d, want 0 because the model never produced a reply", messageRepository.assistantCalls)
	}
	if turnLeaseRepository.releaseCalls != 1 || turnLeaseRepository.lastRelease.Status != service.TurnLeaseFailed {
		t.Fatalf("release calls=%d status=%v, want 1 release with failed status", turnLeaseRepository.releaseCalls, turnLeaseRepository.lastRelease.Status)
	}
}

// 未配置 API Key 时的兜底回复必须跟正常回复走同一套落库规则：持久化成功才发 done，
// 否则刷新页面后这一轮 assistant 回复会从历史里消失。
func TestChatStreamHandlerPersistsFallbackReplyWhenModelNotConfigured(t *testing.T) {
	messageRepository := &handlerMessageRepository{
		result: service.AppendUserMessageResult{
			Message: service.UserMessage{ID: 42},
			Created: true,
		},
	}
	chatModel := &handlerChatModel{streamErr: llm.ErrNotConfigured}
	turnLeaseRepository := &handlerTurnLeaseRepository{acquireResult: service.AcquireTurnLeaseResult{Acquired: true}}
	server := newChatHandlerTestServerWithLease(messageRepository, chatModel, turnLeaseRepository)

	recorder := performChatRequest(server, `{
		"session_id":"session_0123456789abcdef0123456789abcdef",
		"client_message_id":"550e8400-e29b-41d4-a716-446655440000",
		"message":"hello"
	}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "event: done") {
		t.Fatalf("body = %q, want done event", recorder.Body.String())
	}
	if turnLeaseRepository.completeCalls != 1 {
		t.Fatalf("complete calls = %d, want 1: fallback must complete atomically", turnLeaseRepository.completeCalls)
	}
	if !strings.Contains(turnLeaseRepository.lastComplete.Content, "未配置大模型 API Key") {
		t.Fatalf("assistant content = %q, want fallback persisted verbatim", turnLeaseRepository.lastComplete.Content)
	}
	if turnLeaseRepository.releaseCalls != 0 {
		t.Fatalf("release calls=%d, want 0 after successful Complete", turnLeaseRepository.releaseCalls)
	}
}

// 落库失败时，即使是兜底回复也不能发 done——避免客户端以为这一轮已经成功保存。
func TestChatStreamHandlerSendsErrorWhenFallbackReplyPersistenceFails(t *testing.T) {
	messageRepository := &handlerMessageRepository{
		result: service.AppendUserMessageResult{
			Message: service.UserMessage{ID: 42},
			Created: true,
		},
	}
	chatModel := &handlerChatModel{streamErr: llm.ErrNotConfigured}
	turnLeaseRepository := &handlerTurnLeaseRepository{
		acquireResult: service.AcquireTurnLeaseResult{Acquired: true},
		completeErr:   errors.New("database unavailable"),
	}
	server := newChatHandlerTestServerWithLease(messageRepository, chatModel, turnLeaseRepository)

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
	if turnLeaseRepository.releaseCalls != 1 || turnLeaseRepository.lastRelease.Status != service.TurnLeaseFailed {
		t.Fatalf("release calls=%d status=%v, want 1 release with failed status", turnLeaseRepository.releaseCalls, turnLeaseRepository.lastRelease.Status)
	}
}

func newChatHandlerTestServer(messageRepository service.MessageRepository, chatModel service.ChatModel) *Server {
	return newChatHandlerTestServerWithLease(messageRepository, chatModel, &handlerTurnLeaseRepository{
		acquireResult: service.AcquireTurnLeaseResult{Acquired: true},
	})
}

func newChatHandlerTestServerWithLease(messageRepository service.MessageRepository, chatModel service.ChatModel, turnLeaseRepository service.TurnLeaseRepository) *Server {
	sessionRepository := &handlerSessionRepository{owners: map[string]string{
		"session_0123456789abcdef0123456789abcdef": "usr_owner",
	}}
	return &Server{
		chat:       service.NewChatService(chatModel, service.DefaultMaxReplyChars),
		sessions:   service.NewSessionService(sessionRepository),
		messages:   service.NewMessageService(messageRepository),
		turnLeases: service.NewTurnLeaseService(turnLeaseRepository),
		log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
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
