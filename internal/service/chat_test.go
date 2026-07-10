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

	err := chatService.Stream(context.Background(), "hello", func(delta string) error {
		reply += delta
		return nil
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if reply != "reply" {
		t.Fatalf("reply = %q, want reply", reply)
	}
	if len(model.messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(model.messages))
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
	if model.messages[1] != (llm.Message{Role: "user", Content: "hello"}) {
		t.Fatalf("user message = %#v", model.messages[1])
	}
}
