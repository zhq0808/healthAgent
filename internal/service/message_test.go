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
