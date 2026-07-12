// Package service 编排身份、会话和聊天等业务用例。
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"healthAgent/internal/llm"
)

// DefaultMaxReplyChars 是未显式配置时 assistant 单条回复累积的最大字符数上限。
// 用来防止模型异常（例如陷入重复输出）时无限占用内存，并避免写入一条超大的数据库行。
const DefaultMaxReplyChars = 4000

// truncationNotice 附加在被截断的回复末尾，让用户和落库内容都清楚这段话没有完整生成。
const truncationNotice = "\n\n（回复过长，已截断）"

// ChatModel 是聊天服务需要的最小模型能力。
type ChatModel interface {
	Timeout() time.Duration
	ModelName() string
	Stream(ctx context.Context, messages []llm.Message, onDelta func(delta string) error) error
}

// ChatService 编排聊天上下文和模型调用。
type ChatService struct {
	model         ChatModel
	prompt        *ChatPrompt
	maxReplyChars int
}

// NewChatService 构造聊天服务。Prompt 必须已在启动期加载并校验。
func NewChatService(model ChatModel, prompt *ChatPrompt, maxReplyChars int) *ChatService {
	if maxReplyChars <= 0 {
		maxReplyChars = DefaultMaxReplyChars
	}
	return &ChatService{model: model, prompt: prompt, maxReplyChars: maxReplyChars}
}

func (s *ChatService) Timeout() time.Duration {
	return s.model.Timeout()
}

func (s *ChatService) PromptVersion() string {
	return s.prompt.Version()
}

func (s *ChatService) ModelName() string {
	return s.model.ModelName()
}

// Stream 组装 system prompt 和服务端读取的可信会话历史，流式调用模型，并把完整回复内容攒好返回。
//
// onDelta 只负责把每段增量转发给调用方（handler 再转成 SSE 帧），不承担任何累积/截断逻辑——
// 这些属于业务规则，由 service 统一负责，避免 handler 里堆业务判断。
// 达到 maxReplyChars 上限时，附加一段截断提示后正常收尾（返回 nil error），
// 因为客户端已经看到了前面这部分内容，不应该被当作一次调用失败。
func (s *ChatService) Stream(ctx context.Context, history []ConversationMessage, onDelta func(delta string) error) (string, error) {
	systemPrompt, err := s.prompt.Render("")
	if err != nil {
		return "", fmt.Errorf("构建 system prompt 失败: %w", err)
	}
	messages := make([]llm.Message, 0, len(history)+1)
	messages = append(messages, llm.Message{Role: "system", Content: systemPrompt})
	for _, message := range history {
		messages = append(messages, llm.Message{Role: message.Role, Content: message.Content})
	}

	var content []byte
	charCount := 0
	truncated := false
	err = s.model.Stream(ctx, messages, func(delta string) error {
		deltaRunes := []rune(delta)
		remaining := s.maxReplyChars - charCount
		if len(deltaRunes) > remaining {
			if remaining > 0 {
				allowed := string(deltaRunes[:remaining])
				content = append(content, allowed...)
				charCount += remaining
				if err := onDelta(allowed); err != nil {
					return err
				}
			}
			truncated = true
			return errReplyTruncated
		}
		content = append(content, delta...)
		charCount += len(deltaRunes)
		return onDelta(delta)
	})
	if errors.Is(err, errReplyTruncated) {
		err = nil
	}
	if err != nil {
		return string(content), err
	}
	if truncated {
		if notifyErr := onDelta(truncationNotice); notifyErr != nil {
			return string(content), notifyErr
		}
		content = append(content, truncationNotice...)
	}
	return string(content), nil
}

// errReplyTruncated 只在 Stream 内部用来打断 model.Stream 的读取循环，从不对外暴露。
var errReplyTruncated = errors.New("assistant 回复已达到最大长度上限")
