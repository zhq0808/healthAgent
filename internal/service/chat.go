// Package service 编排身份、会话和聊天等业务用例。
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"healthAgent/internal/llm"
)

// DefaultMaxReplyChars 是未显式配置时 assistant 单条回复累积的最大字符数上限。
// 用来防止模型异常（例如陷入重复输出）时无限占用内存，并避免写入一条超大的数据库行。
const DefaultMaxReplyChars = 4000

// truncationNotice 附加在被截断的回复末尾，让用户和落库内容都清楚这段话没有完整生成。
const truncationNotice = "\n\n（回复过长，已截断）"

// memoryContextItemPrefix 是每条已确认记忆在数据块里的固定前缀，配合类型标签构成固定格式。
const memoryContextItemPrefix = "- "

// memoryOmittedNoticeFormat 在因长度预算丢弃记忆时补一行可见提示，避免“静默截断”让模型误以为召回已完整。
const memoryOmittedNoticeFormat = "（另有 %d 条已确认记忆因长度预算未展示，如需完整信息请向用户确认）"

// memoryValueNeutralizer 把记忆内容里的换行、回车、制表折叠成空格。
// 记忆内容来自历史对话，若保留换行，一条记忆就能伪造出新的段落标题或“系统指令行”，
// 从而在 Prompt 里冒充更高优先级的约束。折叠成单行后，memory_value 只能作为一条背景数据存在。
var memoryValueNeutralizer = strings.NewReplacer(
	"\r\n", " ",
	"\r", " ",
	"\n", " ",
	"\t", " ",
)

// ChatModel 是聊天服务需要的最小模型能力。
type ChatModel interface {
	Timeout() time.Duration
	ModelName() string
	Stream(ctx context.Context, messages []llm.Message, onDelta func(delta string) error) error
}

// ChatMemoryReader 提供当前用户跨 Session 的已确认长期记忆。
type ChatMemoryReader interface {
	ListCurrentMemories(ctx context.Context, userID string, budget MemoryBudget) ([]Memory, error)
}

// ChatService 编排聊天上下文和模型调用。
type ChatService struct {
	model         ChatModel
	prompt        *ChatPrompt
	memories      ChatMemoryReader
	memoryBudget  MemoryBudget
	maxReplyChars int
}

// NewChatService 构造聊天服务。Prompt 必须已在启动期加载并校验。
func NewChatService(model ChatModel, prompt *ChatPrompt, memories ChatMemoryReader, memoryBudget MemoryBudget, maxReplyChars int) *ChatService {
	if maxReplyChars <= 0 {
		maxReplyChars = DefaultMaxReplyChars
	}
	return &ChatService{
		model:         model,
		prompt:        prompt,
		memories:      memories,
		memoryBudget:  memoryBudget,
		maxReplyChars: maxReplyChars,
	}
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
func (s *ChatService) Stream(ctx context.Context, userID string, history []ConversationMessage, onDelta func(delta string) error) (string, error) {
	userProfileSummary, err := s.loadUserProfileSummary(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("加载用户记忆失败: %w", err)
	}
	systemPrompt, err := s.prompt.Render(userProfileSummary)
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

func (s *ChatService) loadUserProfileSummary(ctx context.Context, userID string) (string, error) {
	if s.memories == nil {
		return "", nil
	}
	// 数量硬上限下推到存储层 LIMIT；字符预算留在渲染层按整行累加，
	// 让丢弃发生在展示层，才能补一行可见提示，避免存储层静默截断关键安全信息。
	memories, err := s.memories.ListCurrentMemories(ctx, userID, MemoryBudget{MaxCount: s.memoryBudget.MaxCount})
	if err != nil {
		return "", err
	}

	lines := make([]string, 0, len(memories))
	for _, memory := range memories {
		if line := renderMemoryLine(memory); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return "", nil
	}

	kept := make([]string, 0, len(lines))
	used := 0
	for _, line := range lines {
		cost := utf8.RuneCountInString(line)
		// 至少保留最新一条完整记忆；不切断单条内容，避免丢掉过敏、禁忌等关键安全信息的后半段。
		if len(kept) > 0 && s.memoryBudget.MaxChars > 0 && used+cost > s.memoryBudget.MaxChars {
			break
		}
		kept = append(kept, line)
		used += cost
	}

	if omitted := len(lines) - len(kept); omitted > 0 {
		kept = append(kept, fmt.Sprintf(memoryOmittedNoticeFormat, omitted))
	}
	return strings.Join(kept, "\n"), nil
}

// renderMemoryLine 把一条已确认记忆渲染成固定格式的单行背景数据；空内容返回空串以跳过。
//
// 记忆内容来自历史对话，只能作为背景事实呈现，不能当成可执行指令：
//   - 加类型标签构成固定格式，便于模型区分数据与指令。
//   - 折叠换行/制表，避免一条记忆伪造出新的“系统指令行”冒充更高优先级约束。
func renderMemoryLine(memory Memory) string {
	value := strings.TrimSpace(memoryValueNeutralizer.Replace(memory.MemoryValue))
	if value == "" {
		return ""
	}
	return fmt.Sprintf("%s[%s] %s", memoryContextItemPrefix, memoryTypeLabel(memory.MemoryType), value)
}

// memoryTypeLabel 把记忆类型归一到允许集合；未知或缺失时统一标为 other，避免把脏标签直接写进 Prompt。
func memoryTypeLabel(memoryType string) string {
	memoryType = strings.TrimSpace(memoryType)
	if _, ok := allowedMemoryTypes[memoryType]; ok {
		return memoryType
	}
	return "other"
}

// errReplyTruncated 只在 Stream 内部用来打断 model.Stream 的读取循环，从不对外暴露。
var errReplyTruncated = errors.New("assistant 回复已达到最大长度上限")
