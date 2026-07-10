package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"healthAgent/internal/llm"
)

type fakeChatModel struct {
	messages []llm.Message
}

func (f *fakeChatModel) Timeout() time.Duration {
	return time.Second
}

func (f *fakeChatModel) Stream(_ context.Context, messages []llm.Message, onDelta func(string) error) error {
	f.messages = messages
	return onDelta("reply")
}

func TestChatServiceStreamBuildsMessages(t *testing.T) {
	model := &fakeChatModel{}
	chatService := NewChatService(model)
	var reply string
	history := []ConversationMessage{
		{Seq: 1, Role: "user", Content: "my name is Alice"},
		{Seq: 2, Role: "assistant", Content: "nice to meet you"},
		{Seq: 3, Role: "user", Content: "what is my name?"},
	}

	err := chatService.Stream(context.Background(), history, func(delta string) error {
		reply += delta
		return nil
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if reply != "reply" {
		t.Fatalf("reply = %q, want reply", reply)
	}
	if len(model.messages) != 4 {
		t.Fatalf("message count = %d, want 4", len(model.messages))
	}
	if model.messages[0].Role != "system" || model.messages[0].Content != systemPrompt {
		t.Fatalf("system message = %#v", model.messages[0])
	}
	if !strings.Contains(model.messages[0].Content, "健康管家") {
		t.Fatalf("system prompt does not define the health assistant role: %q", model.messages[0].Content)
	}
	if strings.Contains(model.messages[0].Content, "Intent Categories") || strings.Contains(model.messages[0].Content, "Strict JSON") {
		t.Fatalf("chat system prompt contains intent-classifier instructions: %q", model.messages[0].Content)
	}
	for index, message := range history {
		want := llm.Message{Role: message.Role, Content: message.Content}
		if model.messages[index+1] != want {
			t.Fatalf("model message %d = %#v, want %#v", index+1, model.messages[index+1], want)
		}
	}
}
