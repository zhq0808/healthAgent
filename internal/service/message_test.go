package service

import (
	"context"
	"errors"
	"testing"
)

type fakeMessageRepository struct {
	result AppendUserMessageResult
	err    error
}

func (r *fakeMessageRepository) AppendUserMessage(_ context.Context, _ AppendUserMessageRequest) (AppendUserMessageResult, error) {
	return r.result, r.err
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
