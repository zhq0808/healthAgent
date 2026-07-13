package service

import (
	"context"
	"errors"
	"testing"
)

type fakeMessageRepository struct {
	result          AppendUserMessageResult
	assistantResult AssistantMessage
	history         []ConversationMessage
	reply           AssistantMessage
	replyFound      bool
	lastFindReplyID string
	sessionMessages []SessionMessage
	err             error
}

func (r *fakeMessageRepository) AppendUserMessage(_ context.Context, _ AppendUserMessageRequest) (AppendUserMessageResult, error) {
	return r.result, r.err
}

func (r *fakeMessageRepository) AppendAssistantMessage(_ context.Context, _ AppendAssistantMessageRequest) (AssistantMessage, error) {
	return r.assistantResult, r.err
}

func (r *fakeMessageRepository) LoadRecent(_ context.Context, _, _ string, _ int) ([]ConversationMessage, error) {
	return r.history, r.err
}

func (r *fakeMessageRepository) FindAssistantReplyByID(_ context.Context, _, _, messageID string) (AssistantMessage, bool, error) {
	r.lastFindReplyID = messageID
	return r.reply, r.replyFound, r.err
}

func (r *fakeMessageRepository) ListMessages(_ context.Context, _, _ string) ([]SessionMessage, error) {
	return r.sessionMessages, r.err
}

func TestMessageServiceReturnsIdempotentExistingMessage(t *testing.T) {
	repository := &fakeMessageRepository{result: AppendUserMessageResult{
		Message: UserMessage{Content: "same content"},
		Created: false,
	}}
	messageService := NewMessageService(repository)

	result, err := messageService.AppendUserMessage(t.Context(), AppendUserMessageRequest{Content: "same content"})
	if err != nil {
		t.Fatalf("AppendUserMessage() error = %v", err)
	}
	if result.Created {
		t.Fatal("AppendUserMessage() Created = true, want false")
	}
}

func TestMessageServiceRejectsReusedIDWithDifferentContent(t *testing.T) {
	repository := &fakeMessageRepository{result: AppendUserMessageResult{
		Message: UserMessage{Content: "original content"},
		Created: false,
	}}
	messageService := NewMessageService(repository)

	_, err := messageService.AppendUserMessage(t.Context(), AppendUserMessageRequest{Content: "changed content"})
	if !errors.Is(err, ErrClientMessageConflict) {
		t.Fatalf("AppendUserMessage() error = %v, want ErrClientMessageConflict", err)
	}
}

func TestMessageServiceDoesNotQueryHistoryForNonPositiveLimit(t *testing.T) {
	messageService := NewMessageService(&fakeMessageRepository{err: errors.New("should not be called")})

	history, err := messageService.LoadRecent(t.Context(), "usr_owner", "session_owner", 0)
	if err != nil {
		t.Fatalf("LoadRecent() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("LoadRecent() = %+v, want empty", history)
	}
}

func TestMessageServiceRejectsEmptyAssistantMessage(t *testing.T) {
	messageService := NewMessageService(&fakeMessageRepository{})

	_, err := messageService.AppendAssistantMessage(t.Context(), AppendAssistantMessageRequest{})
	if !errors.Is(err, ErrEmptyAssistantMessage) {
		t.Fatalf("AppendAssistantMessage() error = %v, want ErrEmptyAssistantMessage", err)
	}
}

func TestMessageServiceFindReplyForTurnQueriesPersistedResultID(t *testing.T) {
	repository := &fakeMessageRepository{reply: AssistantMessage{Content: "answer"}, replyFound: true}
	messageService := NewMessageService(repository)

	reply, found, err := messageService.FindReplyForTurn(t.Context(), "usr_owner", "session_owner", "msg-99")
	if err != nil {
		t.Fatalf("FindReplyForTurn() error = %v", err)
	}
	if !found || reply.Content != "answer" {
		t.Fatalf("FindReplyForTurn() = %+v, %v, want found answer", reply, found)
	}
	if repository.lastFindReplyID != "msg-99" {
		t.Fatalf("queried message id = %q, want msg-99", repository.lastFindReplyID)
	}
}

func TestMessageServiceFindReplyForTurnReportsNotFound(t *testing.T) {
	messageService := NewMessageService(&fakeMessageRepository{replyFound: false})

	_, found, err := messageService.FindReplyForTurn(t.Context(), "usr_owner", "session_owner", "msg-5")
	if err != nil {
		t.Fatalf("FindReplyForTurn() error = %v", err)
	}
	if found {
		t.Fatal("FindReplyForTurn() found = true, want false")
	}
}

func TestMessageServiceListMessagesReturnsRepositoryResult(t *testing.T) {
	repository := &fakeMessageRepository{sessionMessages: []SessionMessage{
		{MessageID: "m1", Role: "user", Content: "hi", Seq: 1},
		{MessageID: "m2", Role: "assistant", Content: "hello", Seq: 2},
	}}
	messageService := NewMessageService(repository)

	messages, err := messageService.ListMessages(t.Context(), "usr_owner", "session_owner")
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("ListMessages() = %+v, want 2 messages", messages)
	}
}

func TestMessageServiceListMessagesPropagatesRepositoryError(t *testing.T) {
	messageService := NewMessageService(&fakeMessageRepository{err: errors.New("database unavailable")})

	if _, err := messageService.ListMessages(t.Context(), "usr_owner", "session_owner"); err == nil {
		t.Fatal("ListMessages() error = nil, want propagated repository error")
	}
}
