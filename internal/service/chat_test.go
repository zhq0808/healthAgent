package service

import (
	"context"
	"errors"
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
	chatService := NewChatService(model, DefaultMaxReplyChars)
	var reply string
	history := []ConversationMessage{
		{Seq: 1, Role: "user", Content: "my name is Alice"},
		{Seq: 2, Role: "assistant", Content: "nice to meet you"},
		{Seq: 3, Role: "user", Content: "what is my name?"},
	}

	content, err := chatService.Stream(context.Background(), history, func(delta string) error {
		reply += delta
		return nil
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if reply != "reply" {
		t.Fatalf("reply = %q, want reply", reply)
	}
	if content != "reply" {
		t.Fatalf("content = %q, want reply", content)
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

// sequencedChatModel 模拟底层模型逐段回调 onDelta ，只要 onDelta 返回 error 就立即停止读取并把该 error 传回去，
// 这与 llm.DeepSeekClient.Stream 的真实行为保持一致。
type sequencedChatModel struct {
	deltas    []string
	streamErr error // 所有 delta 都正常发完后再返回的错误，模拟真实的上游调用失败
}

func (m *sequencedChatModel) Timeout() time.Duration { return time.Second }

func (m *sequencedChatModel) Stream(_ context.Context, _ []llm.Message, onDelta func(string) error) error {
	for _, delta := range m.deltas {
		if err := onDelta(delta); err != nil {
			return err
		}
	}
	return m.streamErr
}

func TestChatServiceStreamTruncatesLongReplyWithoutTreatingItAsFailure(t *testing.T) {
	model := &sequencedChatModel{deltas: []string{"12345", "67890", "abcde"}}
	chatService := NewChatService(model, 5) // 上限 5 个字符，第一段刚好用完

	var forwarded []string
	content, err := chatService.Stream(context.Background(), nil, func(delta string) error {
		forwarded = append(forwarded, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil：截断不算调用失败，客户端已经看到的内容不该被当成一次错误", err)
	}
	if !strings.HasPrefix(content, "12345") {
		t.Fatalf("content = %q, want to start with the first chunk that filled the cap", content)
	}
	if !strings.Contains(content, "已截断") {
		t.Fatalf("content = %q, want a truncation notice appended", content)
	}
	if len(forwarded) != 2 || forwarded[0] != "12345" || forwarded[1] != truncationNotice {
		t.Fatalf("forwarded = %v, want exactly [12345, truncationNotice]；超出上限后的分片不应再转发给客户端", forwarded)
	}
}

func TestChatServiceStreamPropagatesRealModelErrorWithPartialContent(t *testing.T) {
	wantErr := errors.New("upstream failed")
	model := &sequencedChatModel{deltas: []string{"partial"}, streamErr: wantErr}
	chatService := NewChatService(model, DefaultMaxReplyChars)

	content, err := chatService.Stream(context.Background(), nil, func(string) error { return nil })
	if !errors.Is(err, wantErr) {
		t.Fatalf("Stream() error = %v, want %v", err, wantErr)
	}
	if content != "partial" {
		t.Fatalf("content = %q, want partial content preserved even when the model call ultimately fails", content)
	}
}

func TestNewChatServiceAppliesDefaultMaxReplyCharsWhenNotPositive(t *testing.T) {
	model := &sequencedChatModel{deltas: []string{strings.Repeat("a", DefaultMaxReplyChars)}}
	chatService := NewChatService(model, 0)

	content, err := chatService.Stream(context.Background(), nil, func(string) error { return nil })
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if strings.Contains(content, "已截断") {
		t.Fatalf("content 被截断，want maxReplyChars <= 0 时回退为 DefaultMaxReplyChars 且刚好不截断: %q", content)
	}
}
