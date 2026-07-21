package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

// MemoryAgentID 是当前唯一的业务 Agent，记忆全部归属它。
const MemoryAgentID = InterviewAgentID

// 记忆操作类型，对应抽取 LLM 契约与 agent_memory_history.action。
const (
	MemoryActionAdd    = "ADD"
	MemoryActionUpdate = "UPDATE"
	MemoryActionDelete = "DELETE"

	maxEvidenceQuoteChars = 1000
)

// allowedMemoryTypes 是 agent_memory_meta.memory_type 允许的容错标签集合；无法判断时用 other。
var allowedMemoryTypes = map[string]struct{}{
	"preference": {},
	"goal":       {},
	"habit":      {},
	"context":    {},
	"other":      {},
}

// 记忆存储/事务相关的领域错误。抽取整批要么全部应用要么整批拒绝，因此这些错误都表示“本批放弃”。
var (
	// ErrMemoryInvalidOperation 表示模型输出结构非法（动作、枚举、数量、字段约束等确定性校验失败）。
	ErrMemoryInvalidOperation = errors.New("记忆操作结构非法")
	// ErrMemoryInvalidReference 表示操作引用了本批输入里不存在的临时引用（M/N）。
	ErrMemoryInvalidReference = errors.New("记忆操作引用了不存在的临时引用")
	// ErrMemorySourceForbidden 表示来源不是本批 user 消息，或来源消息不属于当前用户。
	ErrMemorySourceForbidden = errors.New("记忆来源不属于本批用户消息")
	// ErrMemoryEvidenceMismatch 表示 evidence_quote 不是来源消息原文的真实子串。
	ErrMemoryEvidenceMismatch = errors.New("evidence_quote 不是来源消息原文的子串")
	// ErrMemoryVersionConflict 表示乐观锁版本不匹配或同一版本重复写入，旧结果不能覆盖新版本。
	ErrMemoryVersionConflict = errors.New("记忆版本冲突，旧结果不能覆盖新版本")
	// ErrExtractionCursorConflict 表示抽取游标或租约已失效，本次提交被拒绝（旧执行者晚到）。
	ErrExtractionCursorConflict = errors.New("抽取游标或租约已失效")
)

// Memory 是一条当前有效的原子记忆快照（对应 agent_memory_meta 未删除行）。
type Memory struct {
	MemoryID              string
	UserID                string
	AgentID               string
	MemoryType            string
	MemoryValue           string
	Confidence            *float64
	LatestSourceMessageID string
	Version               int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// MemoryBudget 约束一次召回加载多少条当前记忆、总字符数上限。MaxCount<=0 或 MaxChars<=0 表示该维度不限。
type MemoryBudget struct {
	MaxCount int
	MaxChars int
}

// ExistingMemoryRef 把一个临时引用（M1、M2……）绑定到一条真实的当前记忆。
// 真实 memory_id / version 只存在于后端，不交给 LLM，防止模型抄错或伪造 ID。
type ExistingMemoryRef struct {
	Ref    string
	Memory Memory
}

// BatchMessageRef 把一个临时引用（N1、N2……）绑定到本批的一条真实消息。
type BatchMessageRef struct {
	Ref       string
	MessageID string
	Role      string
	Content   string
	Seq       int64
}

// LLMMemorySource 是抽取 LLM 为某个操作给出的一条来源引用与原文证据。
// 只带临时引用与原文片段，不含真实 message_id。
type LLMMemorySource struct {
	Ref           string `json:"ref"`
	EvidenceQuote string `json:"evidence_quote"`
}

// LLMMemoryOperation 是抽取 LLM 返回的一条操作。它只携带临时引用与自然语言内容，
// 刻意不含 user_id / memory_id / message_id / version / 时间字段——这些只能由后端可信上下文维护。
// 配合 ParseExtractionOperations 的 DisallowUnknownFields，模型伪造这些字段会被直接拒绝。
type LLMMemoryOperation struct {
	Action       string            `json:"action"`
	TargetRef    string            `json:"target_ref"`
	Sources      []LLMMemorySource `json:"sources"`
	MemoryType   string            `json:"memory_type"`
	MemoryValue  string            `json:"memory_value"`
	Explicitness string            `json:"explicitness"`
	Confidence   *float64          `json:"confidence"`
}

// ResolvedSource 是一条已解析、已验证的来源：临时引用换成真实 message_id，原文片段已确认是子串。
type ResolvedSource struct {
	SourceOrder   int16
	MessageID     string
	EvidenceQuote string
}

// ResolvedOperation 是通过全部确定性校验、临时引用已换成真实 ID 的一条可落库操作。
// ADD 时 MemoryID 为空（由存储层生成 UUIDv7），ExpectedVersion 为 0；
// UPDATE/DELETE 时 MemoryID/ExpectedVersion 来自后端已知的当前记忆，供乐观锁使用。
type ResolvedOperation struct {
	Action                string
	MemoryID              string
	ExpectedVersion       int64
	MemoryType            string
	MemoryValue           string
	Confidence            *float64
	Sources               []ResolvedSource
	LatestSourceMessageID string
}

// ApplyExtractionRequest 是存储层在一个短事务里落库所需的可信数据：
// 已解析操作 + 抽取元数据 + 游标条件提交（from/to + lease_token）。
type ApplyExtractionRequest struct {
	UserID           string
	AgentID          string
	SessionID        string
	Operations       []ResolvedOperation
	ExtractorModel   string
	ExtractorVersion string
	FromResultSeq    int64
	ToResultSeq      int64
	LeaseToken       string
}

// ApplyExtractionResult 汇报本次提交结果：新增/更新/删除条数与推进后的游标。
type ApplyExtractionResult struct {
	Added       int
	Updated     int
	Deleted     int
	ToResultSeq int64
}

// MemoryRepository 是记忆存储所需的最小持久化能力；短事务与并发裁决在实现里完成，handler 不直接拼 SQL。
type MemoryRepository interface {
	// ListCurrentMemories 按 user_id + agent_id 返回未删除的当前记忆，按更新时间倒序，受数量预算约束。
	ListCurrentMemories(ctx context.Context, userID, agentID string, maxCount int) ([]Memory, error)
	// ApplyExtraction 在一个短事务内应用全部操作并条件推进抽取游标；任一步失败整体回滚。
	ApplyExtraction(ctx context.Context, request ApplyExtractionRequest) (ApplyExtractionResult, error)
}

// MemoryExtractionLimits 是一次抽取应用的硬上限，来自配置，避免无界增长与费用失控。
type MemoryExtractionLimits struct {
	MaxOperations       int     // 单批最多操作数
	MaxMemoryValueChars int     // 单条记忆最大字符数
	MinConfidence       float64 // 可应用的最低置信度
}

// DefaultMemoryExtractionLimits 是首版默认上限，最终阈值由独立语义评测校准。
var DefaultMemoryExtractionLimits = MemoryExtractionLimits{
	MaxOperations:       20,
	MaxMemoryValueChars: 500,
	MinConfidence:       0.6,
}

// MemoryService 编排记忆召回与抽取结果的确定性校验/落库。它不重新理解自然语言，
// 只做“把临时引用换成真实 ID + 校验确定性事实 + 短事务落库”。
type MemoryService struct {
	repository MemoryRepository
	limits     MemoryExtractionLimits
}

// NewMemoryService 构造记忆服务。limits 各项 <=0 时回退到默认上限，避免误配成“不限”。
func NewMemoryService(repository MemoryRepository, limits MemoryExtractionLimits) *MemoryService {
	if limits.MaxOperations <= 0 {
		limits.MaxOperations = DefaultMemoryExtractionLimits.MaxOperations
	}
	if limits.MaxMemoryValueChars <= 0 {
		limits.MaxMemoryValueChars = DefaultMemoryExtractionLimits.MaxMemoryValueChars
	}
	if limits.MinConfidence <= 0 {
		limits.MinConfidence = DefaultMemoryExtractionLimits.MinConfidence
	}
	return &MemoryService{repository: repository, limits: limits}
}

// ListCurrentMemories 返回当前用户当前 agent 的有效记忆，并遵守数量/字符预算：
// 数量预算下推到存储层 LIMIT；字符预算在应用层按顺序累加，超预算即停止，不静默截断单条记忆内容。
func (s *MemoryService) ListCurrentMemories(ctx context.Context, userID string, budget MemoryBudget) ([]Memory, error) {
	if userID == "" {
		return nil, errors.New("加载记忆缺少用户标识")
	}
	memories, err := s.repository.ListCurrentMemories(ctx, userID, MemoryAgentID, budget.MaxCount)
	if err != nil {
		return nil, err
	}
	if budget.MaxChars <= 0 {
		return memories, nil
	}
	used := 0
	kept := make([]Memory, 0, len(memories))
	for _, memory := range memories {
		cost := utf8.RuneCountInString(memory.MemoryValue)
		if len(kept) > 0 && used+cost > budget.MaxChars {
			break
		}
		kept = append(kept, memory)
		used += cost
	}
	return kept, nil
}

// extractionOutput 使用指针区分 operations 缺失/null 与合法空数组。
type extractionOutput struct {
	Operations *[]extractionWireOperation `json:"operations"`
}

// extractionWireOperation 仅用于严格解析模型 JSON。指针字段用于识别必填字段缺失/null；
// 通过解析后再转换成不含解析细节的领域类型 LLMMemoryOperation。
type extractionWireOperation struct {
	Action       *string                 `json:"action"`
	TargetRef    *string                 `json:"target_ref"`
	Sources      *[]extractionWireSource `json:"sources"`
	MemoryType   *string                 `json:"memory_type"`
	MemoryValue  *string                 `json:"memory_value"`
	Explicitness *string                 `json:"explicitness"`
	Confidence   *float64                `json:"confidence"`
}

type extractionWireSource struct {
	Ref           *string `json:"ref"`
	EvidenceQuote *string `json:"evidence_quote"`
}

// ParseExtractionOperations 严格解析抽取 LLM 的 JSON 输出：拒绝未知字段（含伪造的 user_id/memory_id 等），
// 拒绝多余的尾随内容。解析失败统一归为 ErrMemoryInvalidOperation，交给调用方走整批拒绝/重试。
func ParseExtractionOperations(raw []byte) ([]LLMMemoryOperation, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var output extractionOutput
	if err := decoder.Decode(&output); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMemoryInvalidOperation, err)
	}
	if output.Operations == nil {
		return nil, fmt.Errorf("%w: operations 缺失或为 null", ErrMemoryInvalidOperation)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: JSON 存在多余尾随内容", ErrMemoryInvalidOperation)
	}

	operations := make([]LLMMemoryOperation, 0, len(*output.Operations))
	for operationIndex, wireOperation := range *output.Operations {
		if wireOperation.Action == nil || wireOperation.TargetRef == nil || wireOperation.Sources == nil ||
			wireOperation.MemoryType == nil || wireOperation.MemoryValue == nil || wireOperation.Explicitness == nil ||
			wireOperation.Confidence == nil {
			return nil, fmt.Errorf("%w: operation[%d] 缺少必填字段或字段为 null", ErrMemoryInvalidOperation, operationIndex)
		}
		sources := make([]LLMMemorySource, 0, len(*wireOperation.Sources))
		for sourceIndex, wireSource := range *wireOperation.Sources {
			if wireSource.Ref == nil || wireSource.EvidenceQuote == nil {
				return nil, fmt.Errorf("%w: operation[%d].sources[%d] 缺少必填字段或字段为 null", ErrMemoryInvalidOperation, operationIndex, sourceIndex)
			}
			sources = append(sources, LLMMemorySource{Ref: *wireSource.Ref, EvidenceQuote: *wireSource.EvidenceQuote})
		}
		operations = append(operations, LLMMemoryOperation{
			Action:       *wireOperation.Action,
			TargetRef:    *wireOperation.TargetRef,
			Sources:      sources,
			MemoryType:   *wireOperation.MemoryType,
			MemoryValue:  *wireOperation.MemoryValue,
			Explicitness: *wireOperation.Explicitness,
			Confidence:   wireOperation.Confidence,
		})
	}
	return operations, nil
}

// ApplyExtractionInput 是把一批抽取结果落库的全部输入：可信身份 + M/N 映射 + 模型操作 + 游标条件。
type ApplyExtractionInput struct {
	UserID           string
	SessionID        string
	ExistingMemories []ExistingMemoryRef
	BatchMessages    []BatchMessageRef
	Operations       []LLMMemoryOperation
	ExtractorModel   string
	ExtractorVersion string
	FromResultSeq    int64
	ToResultSeq      int64
	LeaseToken       string
}

// ApplyExtraction 对一批抽取结果做全部确定性校验、把临时引用换成真实 ID，然后在一个短事务里落库并推进游标。
//
// 关键安全边界（对应详设第 9/10 节）：
//   - LLM 只提供临时引用 M/N 和自然语言内容；真实 user_id/memory_id/message_id/version 全部由后端解析。
//   - 低置信度或非显式操作被过滤（不是错误），过滤后仍推进游标，避免永久重复处理同一批消息。
//   - 结构非法、引用非法、来源越权、原文对不上均整批拒绝，不部分应用，也不推进游标。
func (s *MemoryService) ApplyExtraction(ctx context.Context, input ApplyExtractionInput) (ApplyExtractionResult, error) {
	if input.UserID == "" || input.SessionID == "" || input.LeaseToken == "" {
		return ApplyExtractionResult{}, errors.New("应用抽取结果缺少必要标识")
	}
	if input.ToResultSeq < input.FromResultSeq {
		return ApplyExtractionResult{}, fmt.Errorf("%w: 目标游标小于起始游标", ErrMemoryInvalidOperation)
	}
	if len(input.Operations) > s.limits.MaxOperations {
		return ApplyExtractionResult{}, fmt.Errorf("%w: 操作数 %d 超过单批上限 %d", ErrMemoryInvalidOperation, len(input.Operations), s.limits.MaxOperations)
	}

	existingByRef := make(map[string]Memory, len(input.ExistingMemories))
	for _, item := range input.ExistingMemories {
		if item.Ref == "" || item.Memory.MemoryID == "" {
			return ApplyExtractionResult{}, fmt.Errorf("%w: 已有记忆引用不完整", ErrMemoryInvalidOperation)
		}
		if item.Memory.UserID != input.UserID {
			return ApplyExtractionResult{}, fmt.Errorf("%w: 已有记忆不属于当前用户", ErrMemorySourceForbidden)
		}
		existingByRef[item.Ref] = item.Memory
	}
	batchByRef := make(map[string]BatchMessageRef, len(input.BatchMessages))
	for _, message := range input.BatchMessages {
		if message.Ref == "" || message.MessageID == "" {
			return ApplyExtractionResult{}, fmt.Errorf("%w: 本批消息引用不完整", ErrMemoryInvalidOperation)
		}
		batchByRef[message.Ref] = message
	}

	resolved := make([]ResolvedOperation, 0, len(input.Operations))
	seenTargets := make(map[string]struct{}, len(input.Operations))
	for _, operation := range input.Operations {
		if err := s.validateOperationShape(operation); err != nil {
			return ApplyExtractionResult{}, err
		}
		resolvedOp, err := s.resolveOperation(operation, existingByRef, batchByRef, seenTargets)
		if err != nil {
			return ApplyExtractionResult{}, err
		}
		// 只有结构、引用、来源全部合法后，implicit 或低置信度操作才允许被过滤并推进游标。
		if operation.Explicitness != "explicit" || *operation.Confidence < s.limits.MinConfidence {
			continue
		}
		resolved = append(resolved, resolvedOp)
	}

	return s.repository.ApplyExtraction(ctx, ApplyExtractionRequest{
		UserID:           input.UserID,
		AgentID:          MemoryAgentID,
		SessionID:        input.SessionID,
		Operations:       resolved,
		ExtractorModel:   input.ExtractorModel,
		ExtractorVersion: input.ExtractorVersion,
		FromResultSeq:    input.FromResultSeq,
		ToResultSeq:      input.ToResultSeq,
		LeaseToken:       input.LeaseToken,
	})
}

func (s *MemoryService) validateOperationShape(operation LLMMemoryOperation) error {
	if operation.Explicitness != "explicit" && operation.Explicitness != "implicit" {
		return fmt.Errorf("%w: 非法 explicitness %q", ErrMemoryInvalidOperation, operation.Explicitness)
	}
	if operation.Confidence == nil || *operation.Confidence < 0 || *operation.Confidence > 1 {
		return fmt.Errorf("%w: confidence 缺失或超出 [0,1]", ErrMemoryInvalidOperation)
	}
	switch operation.Action {
	case MemoryActionAdd:
		if operation.TargetRef != "" {
			return fmt.Errorf("%w: ADD 不允许 target_ref", ErrMemoryInvalidOperation)
		}
		if err := s.validateMemoryType(operation.MemoryType); err != nil {
			return err
		}
		return s.validateMemoryValue(operation.MemoryValue)
	case MemoryActionUpdate:
		if operation.TargetRef == "" {
			return fmt.Errorf("%w: UPDATE 缺少 target_ref", ErrMemoryInvalidOperation)
		}
		if err := s.validateMemoryType(operation.MemoryType); err != nil {
			return err
		}
		return s.validateMemoryValue(operation.MemoryValue)
	case MemoryActionDelete:
		if operation.TargetRef == "" {
			return fmt.Errorf("%w: DELETE 缺少 target_ref", ErrMemoryInvalidOperation)
		}
		if err := s.validateMemoryType(operation.MemoryType); err != nil {
			return err
		}
		if operation.MemoryValue != "" {
			return fmt.Errorf("%w: DELETE 的 memory_value 必须为空", ErrMemoryInvalidOperation)
		}
		return nil
	default:
		return fmt.Errorf("%w: 未知 action %q", ErrMemoryInvalidOperation, operation.Action)
	}
}

// resolveOperation 校验单条操作并把临时引用换成真实 ID。任何非法情况都返回错误，交给上层整批拒绝。
func (s *MemoryService) resolveOperation(
	operation LLMMemoryOperation,
	existingByRef map[string]Memory,
	batchByRef map[string]BatchMessageRef,
	seenTargets map[string]struct{},
) (ResolvedOperation, error) {
	sources, latestSourceMessageID, err := s.resolveSources(operation.Sources, batchByRef)
	if err != nil {
		return ResolvedOperation{}, err
	}

	switch operation.Action {
	case MemoryActionAdd:
		if operation.TargetRef != "" {
			return ResolvedOperation{}, fmt.Errorf("%w: ADD 不允许 target_ref", ErrMemoryInvalidOperation)
		}
		if err := s.validateMemoryType(operation.MemoryType); err != nil {
			return ResolvedOperation{}, err
		}
		if err := s.validateMemoryValue(operation.MemoryValue); err != nil {
			return ResolvedOperation{}, err
		}
		return ResolvedOperation{
			Action:                MemoryActionAdd,
			MemoryType:            operation.MemoryType,
			MemoryValue:           operation.MemoryValue,
			Confidence:            operation.Confidence,
			Sources:               sources,
			LatestSourceMessageID: latestSourceMessageID,
		}, nil

	case MemoryActionUpdate:
		target, err := s.resolveTarget(operation.TargetRef, existingByRef, seenTargets)
		if err != nil {
			return ResolvedOperation{}, err
		}
		if err := s.validateMemoryType(operation.MemoryType); err != nil {
			return ResolvedOperation{}, err
		}
		if err := s.validateMemoryValue(operation.MemoryValue); err != nil {
			return ResolvedOperation{}, err
		}
		return ResolvedOperation{
			Action:                MemoryActionUpdate,
			MemoryID:              target.MemoryID,
			ExpectedVersion:       target.Version,
			MemoryType:            operation.MemoryType,
			MemoryValue:           operation.MemoryValue,
			Confidence:            operation.Confidence,
			Sources:               sources,
			LatestSourceMessageID: latestSourceMessageID,
		}, nil

	case MemoryActionDelete:
		target, err := s.resolveTarget(operation.TargetRef, existingByRef, seenTargets)
		if err != nil {
			return ResolvedOperation{}, err
		}
		// DELETE 不携带新值；memory_type 沿用当前记忆的标签，仅用于历史留痕。
		return ResolvedOperation{
			Action:                MemoryActionDelete,
			MemoryID:              target.MemoryID,
			ExpectedVersion:       target.Version,
			MemoryType:            target.MemoryType,
			Confidence:            operation.Confidence,
			Sources:               sources,
			LatestSourceMessageID: latestSourceMessageID,
		}, nil

	default:
		return ResolvedOperation{}, fmt.Errorf("%w: 未知 action %q", ErrMemoryInvalidOperation, operation.Action)
	}
}

// resolveTarget 解析 UPDATE/DELETE 的目标已有记忆，并防止同一批重复修改同一条记忆。
func (s *MemoryService) resolveTarget(targetRef string, existingByRef map[string]Memory, seenTargets map[string]struct{}) (Memory, error) {
	if targetRef == "" {
		return Memory{}, fmt.Errorf("%w: UPDATE/DELETE 缺少 target_ref", ErrMemoryInvalidOperation)
	}
	target, ok := existingByRef[targetRef]
	if !ok {
		return Memory{}, fmt.Errorf("%w: target_ref %q", ErrMemoryInvalidReference, targetRef)
	}
	if _, duplicated := seenTargets[target.MemoryID]; duplicated {
		return Memory{}, fmt.Errorf("%w: 同一批重复修改同一条记忆", ErrMemoryInvalidOperation)
	}
	seenTargets[target.MemoryID] = struct{}{}
	return target, nil
}

// resolveSources 校验并解析来源：至少一条、都在本批、都是 user 消息、原文子串可验证，
// 并返回本次变更中 Session 内 seq 最大的 user 来源消息 message_id 作为 latest_source_message_id。
func (s *MemoryService) resolveSources(sources []LLMMemorySource, batchByRef map[string]BatchMessageRef) ([]ResolvedSource, string, error) {
	if len(sources) == 0 {
		return nil, "", fmt.Errorf("%w: 操作缺少来源", ErrMemoryInvalidOperation)
	}
	resolved := make([]ResolvedSource, 0, len(sources))
	latestMessageID := ""
	latestSeq := int64(-1)
	for index, source := range sources {
		message, ok := batchByRef[source.Ref]
		if !ok {
			return nil, "", fmt.Errorf("%w: source ref %q", ErrMemoryInvalidReference, source.Ref)
		}
		if message.Role != "user" {
			return nil, "", fmt.Errorf("%w: 来源 %q 不是 user 消息", ErrMemorySourceForbidden, source.Ref)
		}
		if strings.TrimSpace(source.EvidenceQuote) == "" {
			return nil, "", fmt.Errorf("%w: evidence_quote 为空", ErrMemoryInvalidOperation)
		}
		if utf8.RuneCountInString(source.EvidenceQuote) > maxEvidenceQuoteChars {
			return nil, "", fmt.Errorf("%w: evidence_quote 超过 %d 字符", ErrMemoryInvalidOperation, maxEvidenceQuoteChars)
		}
		if !strings.Contains(message.Content, source.EvidenceQuote) {
			return nil, "", fmt.Errorf("%w: source ref %q", ErrMemoryEvidenceMismatch, source.Ref)
		}
		resolved = append(resolved, ResolvedSource{
			SourceOrder:   int16(index + 1),
			MessageID:     message.MessageID,
			EvidenceQuote: source.EvidenceQuote,
		})
		if message.Seq > latestSeq {
			latestSeq = message.Seq
			latestMessageID = message.MessageID
		}
	}
	return resolved, latestMessageID, nil
}

func (s *MemoryService) validateMemoryType(memoryType string) error {
	if _, ok := allowedMemoryTypes[memoryType]; !ok {
		return fmt.Errorf("%w: 非法 memory_type %q", ErrMemoryInvalidOperation, memoryType)
	}
	return nil
}

func (s *MemoryService) validateMemoryValue(memoryValue string) error {
	if strings.TrimSpace(memoryValue) == "" {
		return fmt.Errorf("%w: memory_value 为空", ErrMemoryInvalidOperation)
	}
	if utf8.RuneCountInString(memoryValue) > s.limits.MaxMemoryValueChars {
		return fmt.Errorf("%w: memory_value 超过 %d 字符", ErrMemoryInvalidOperation, s.limits.MaxMemoryValueChars)
	}
	return nil
}

// NewMemoryID 生成记忆业务主键 UUIDv7。
func NewMemoryID() (string, error) {
	return newUUIDv7("memory_id")
}

// NewHistoryID 生成记忆历史事件 UUIDv7。
func NewHistoryID() (string, error) {
	return newUUIDv7("history_id")
}
