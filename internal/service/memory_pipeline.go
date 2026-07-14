package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

const (
	failureRecordTimeout = 3 * time.Second
	leaseCommitMargin    = 5 * time.Second
)

// MemoryPipelineConfig 是异步抽取管道的运行参数与硬上限。
type MemoryPipelineConfig struct {
	WorkerCount      int           // 固定 Worker 数量，禁止每个 turn 无界起 goroutine
	QueueSize        int           // 有界通知队列容量，满时丢弃通知（补扫兜底）
	ScanInterval     time.Duration // 数据库补扫周期
	LeaseDuration    time.Duration // Session 抽取租约时长，必须大于抽取超时 + 提交余量
	ExtractTimeout   time.Duration // 单次抽取 LLM 调用超时
	TaskTimeout      time.Duration // 单个任务（含读批次/调用/落库）的总预算
	ScanBatchSize    int           // 单次补扫返回的最多 Session 数
	MaxBatchMessages int           // 单批交给抽取器的最多消息数
	MaxBatchChars    int           // 单批消息总字符上限
	MemoryInputLimit int           // 作为 M 引用输入的最多当前记忆条数
	MemoryInputChars int           // 当前记忆输入总字符上限
	BaseRetryBackoff time.Duration // 失败退避基数
	MaxRetryBackoff  time.Duration // 失败退避上限
	ShutdownGrace    time.Duration // 关闭时等待在途任务的有界时间
	ExtractorModel   string        // 记录到 history 的抽取模型名
	ExtractorVersion string        // 记录到 history 的抽取版本
}

// memoryPipelineMetrics 是管道的轻量运行指标（原子计数），不含任何健康原文。
type memoryPipelineMetrics struct {
	dropped   atomic.Int64 // 队列满/关闭中丢弃的通知数
	processed atomic.Int64 // 成功应用（含空操作推进游标）的任务数
	failed    atomic.Int64 // 记为失败并退避的任务数
	skipped   atomic.Int64 // 无积压/被他人持有/租约被接管而跳过的任务数
}

// MemoryPipelineMetrics 是对外只读的指标快照。
type MemoryPipelineMetrics struct {
	Dropped   int64
	Processed int64
	Failed    int64
	Skipped   int64
}

// MemoryPipeline 编排异步记忆抽取：有界通知队列 + 固定 Worker Pool + 定时补扫 +
// 用户 keyed lock + Session 租约 + 失败退避 + 有界优雅关闭。
//
// 聊天主链路只调用 Notify 做非阻塞投递；真正可恢复的进度落在数据库抽取游标/租约上，
// 通知丢失或进程重启后由补扫兜底。
type MemoryPipeline struct {
	memories   MemoryAccess
	extraction ExtractionRepository
	extractor  MemoryExtractor
	locks      *KeyedMutex
	cfg        MemoryPipelineConfig
	log        *slog.Logger

	notify  chan string
	metrics memoryPipelineMetrics

	startOnce sync.Once
	closeOnce sync.Once
	started   atomic.Bool
	closing   atomic.Bool

	stop     chan struct{}   // 关闭：通知 Worker/补扫停止领取新任务
	hardCtx  context.Context // 关闭宽限用尽后取消，用于中断在途任务
	hardStop context.CancelFunc
	wg       sync.WaitGroup
	closeErr error
}

// NewMemoryPipeline 构造抽取管道。危险的超时/租约关系会直接返回错误，避免带病启动。
func NewMemoryPipeline(memories MemoryAccess, extraction ExtractionRepository, extractor MemoryExtractor, cfg MemoryPipelineConfig, log *slog.Logger) (*MemoryPipeline, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	p := &MemoryPipeline{
		memories:   memories,
		extraction: extraction,
		extractor:  extractor,
		locks:      NewKeyedMutex(),
		cfg:        cfg,
		log:        log,
		notify:     make(chan string, cfg.QueueSize),
		stop:       make(chan struct{}),
	}
	// hardCtx 在构造时即建立：既供 Start 后的在途任务取消，也让 process 可被直接单测。
	p.hardCtx, p.hardStop = context.WithCancel(context.Background())
	return p, nil
}

// Start 启动固定数量 Worker 和一个补扫协程。重复调用无副作用。
func (p *MemoryPipeline) Start() {
	p.startOnce.Do(func() {
		p.started.Store(true)
		for i := 0; i < p.cfg.WorkerCount; i++ {
			p.wg.Add(1)
			go p.worker()
		}
		p.wg.Add(1)
		go p.scanner()
	})
}

// Notify 非阻塞投递一个待抽取 Session；队列满或正在关闭时丢弃并计数，绝不阻塞聊天主链路。
func (p *MemoryPipeline) Notify(sessionID string) {
	if p.closing.Load() {
		p.metrics.dropped.Add(1)
		return
	}
	select {
	case p.notify <- sessionID:
	default:
		p.metrics.dropped.Add(1)
		p.log.Debug("记忆抽取通知队列已满，丢弃通知（补扫兜底）", "session_id", sessionID)
	}
}

// Close 优雅关闭：停止接收新任务，等待在途任务在宽限时间内结束，超时则取消在途任务。
func (p *MemoryPipeline) Close() error {
	if !p.started.Load() {
		return nil
	}
	p.closeOnce.Do(func() {
		p.closing.Store(true)
		close(p.stop)

		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		timer := time.NewTimer(p.cfg.ShutdownGrace)
		defer timer.Stop()
		select {
		case <-done:
			p.hardStop()
		case <-timer.C:
			p.hardStop()
			p.closeErr = errors.New("记忆抽取管道关闭超时，已取消在途任务")
		}
	})
	return p.closeErr
}

// Metrics 返回当前运行指标快照。
func (p *MemoryPipeline) Metrics() MemoryPipelineMetrics {
	return MemoryPipelineMetrics{
		Dropped:   p.metrics.dropped.Load(),
		Processed: p.metrics.processed.Load(),
		Failed:    p.metrics.failed.Load(),
		Skipped:   p.metrics.skipped.Load(),
	}
}

// worker 从通知队列取 Session 处理；收到停止信号后处理完当前任务即退出，不领新任务。
func (p *MemoryPipeline) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.stop:
			return
		default:
		}
		select {
		case <-p.stop:
			return
		case sessionID := <-p.notify:
			p.process(sessionID)
		}
	}
}

// scanner 启动时先补扫一次，之后按周期补扫；收到停止信号退出。
func (p *MemoryPipeline) scanner() {
	defer p.wg.Done()
	p.scanOnce()
	ticker := time.NewTicker(p.cfg.ScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.scanOnce()
		}
	}
}

// scanOnce 找出有积压且可执行的 Session，非阻塞投递到通知队列，由 Worker 统一处理。
func (p *MemoryPipeline) scanOnce() {
	ctx, cancel := context.WithTimeout(p.hardCtx, p.cfg.TaskTimeout)
	defer cancel()
	sessions, err := p.extraction.ScanExtractionBacklog(ctx, p.cfg.ScanBatchSize)
	if err != nil {
		p.log.Error("补扫抽取积压失败", "error", err)
		return
	}
	for _, sessionID := range sessions {
		select {
		case p.notify <- sessionID:
		default:
			p.metrics.dropped.Add(1)
		}
	}
}

// process 处理单个 Session 的一次抽取：查用户 → 用户锁 → 抢租约 → 读批次 → 载记忆 → 调抽取器 → 落库。
// 顺序遵循详设：先拿用户 keyed lock 再抢数据库租约，避免持租约排队导致租约过期。
func (p *MemoryPipeline) process(sessionID string) {
	ctx, cancel := context.WithTimeout(p.hardCtx, p.cfg.TaskTimeout)
	defer cancel()

	userID, found, err := p.extraction.LookupSessionUser(ctx, sessionID)
	if err != nil {
		p.log.Error("查询抽取会话用户失败", "session_id", sessionID, "error", err)
		return
	}
	if !found {
		// 会话不存在或已删除：通知丢弃，不当作错误。
		p.metrics.skipped.Add(1)
		return
	}

	release, err := p.locks.Lock(ctx, userID)
	if err != nil {
		p.metrics.skipped.Add(1)
		return
	}
	defer release()

	lease, acquired, err := p.extraction.AcquireExtractionLease(ctx, sessionID, userID, p.cfg.LeaseDuration)
	if err != nil {
		p.log.Error("获取抽取租约失败", "session_id", sessionID, "error", err)
		return
	}
	if !acquired {
		// 已有未过期租约 / 尚未到重试时间 / 无积压。
		p.metrics.skipped.Add(1)
		return
	}

	turns, err := p.extraction.LoadExtractionBatch(ctx, sessionID, userID, lease.FromResultSeq, lease.ToResultSeq)
	if err != nil {
		p.recordFailure(lease, "load_batch")
		return
	}
	if len(turns) == 0 {
		p.recordFailure(lease, "empty_batch")
		return
	}

	includedTurns, effectiveTo := p.capBatch(turns, lease.ToResultSeq)
	batchMessages := buildBatchRefs(includedTurns)

	existing, err := p.memories.ListCurrentMemories(ctx, userID, MemoryBudget{
		MaxCount: p.cfg.MemoryInputLimit,
		MaxChars: p.cfg.MemoryInputChars,
	})
	if err != nil {
		p.recordFailure(lease, "load_memories")
		return
	}
	existingRefs := buildMemoryRefs(existing)

	extractCtx, extractCancel := context.WithTimeout(ctx, p.cfg.ExtractTimeout)
	operations, err := p.extractor.Extract(extractCtx, ExtractionInput{
		ExistingMemories: existingRefs,
		BatchMessages:    batchMessages,
	})
	extractCancel()
	if err != nil {
		p.recordFailure(lease, classifyExtractionError(err))
		return
	}

	_, err = p.memories.ApplyExtraction(ctx, ApplyExtractionInput{
		UserID:           userID,
		SessionID:        sessionID,
		ExistingMemories: existingRefs,
		BatchMessages:    batchMessages,
		Operations:       operations,
		ExtractorModel:   p.cfg.ExtractorModel,
		ExtractorVersion: p.cfg.ExtractorVersion,
		FromResultSeq:    lease.FromResultSeq,
		ToResultSeq:      effectiveTo,
		LeaseToken:       lease.LeaseToken,
	})
	if errors.Is(err, ErrExtractionCursorConflict) {
		// 租约已过期/被接管：由接管者重跑，本次不记失败。
		p.metrics.skipped.Add(1)
		return
	}
	if err != nil {
		p.recordFailure(lease, classifyExtractionError(err))
		return
	}
	p.metrics.processed.Add(1)
}

// recordFailure 记一次失败并按退避设置重试时间；释放/接管由 lease_token 条件保证。
func (p *MemoryPipeline) recordFailure(lease ExtractionLease, code string) {
	p.metrics.failed.Add(1)
	ctx, cancel := context.WithTimeout(context.Background(), failureRecordTimeout)
	defer cancel()
	if err := p.extraction.RecordExtractionFailure(ctx, lease.SessionID, lease.UserID, lease.LeaseToken, code, p.cfg.BaseRetryBackoff, p.cfg.MaxRetryBackoff); err != nil {
		p.log.Error("记录抽取失败退避失败", "session_id", lease.SessionID, "error", err)
	}
}

// capBatch 按消息数/字符预算裁剪本批处理的 turn。若第一轮本身超过字符预算，只裁剪交给
// 抽取模型的副本（优先保留 user 原话），数据库原消息不变，避免该 Session 永久重试同一毒消息。
// 未被本次纳入的后续 turn 由下一次抽取继续处理。
func (p *MemoryPipeline) capBatch(turns []ExtractionTurn, fullTo int64) ([]ExtractionTurn, int64) {
	if len(turns) == 0 {
		return turns, fullTo
	}
	included := make([]ExtractionTurn, 0, len(turns))
	messages := 0
	chars := 0
	for _, turn := range turns {
		turnChars := utf8.RuneCountInString(turn.UserMessage.Content) + utf8.RuneCountInString(turn.AssistantMessage.Content)
		if messages+2 > p.cfg.MaxBatchMessages || chars+turnChars > p.cfg.MaxBatchChars {
			if len(included) == 0 {
				included = append(included, truncateExtractionTurn(turn, p.cfg.MaxBatchChars))
			}
			break
		}
		included = append(included, turn)
		messages += 2
		chars += turnChars
	}
	if len(included) == len(turns) {
		return included, fullTo
	}
	return included, included[len(included)-1].ResultSeq
}

func truncateExtractionTurn(turn ExtractionTurn, maxChars int) ExtractionTurn {
	userRunes := []rune(turn.UserMessage.Content)
	if len(userRunes) >= maxChars {
		turn.UserMessage.Content = string(userRunes[:maxChars])
		turn.AssistantMessage.Content = ""
		return turn
	}
	remaining := maxChars - len(userRunes)
	assistantRunes := []rune(turn.AssistantMessage.Content)
	if len(assistantRunes) > remaining {
		turn.AssistantMessage.Content = string(assistantRunes[:remaining])
	}
	return turn
}

// buildBatchRefs 把 turn 展开成按 seq 顺序的 N 引用消息列表（user、assistant 交替）。
func buildBatchRefs(turns []ExtractionTurn) []BatchMessageRef {
	refs := make([]BatchMessageRef, 0, len(turns)*2)
	index := 1
	for _, turn := range turns {
		user := turn.UserMessage
		user.Ref = fmt.Sprintf("N%d", index)
		index++
		refs = append(refs, user)

		assistant := turn.AssistantMessage
		assistant.Ref = fmt.Sprintf("N%d", index)
		index++
		refs = append(refs, assistant)
	}
	return refs
}

// buildMemoryRefs 给当前记忆分配 M 引用。
func buildMemoryRefs(memories []Memory) []ExistingMemoryRef {
	refs := make([]ExistingMemoryRef, 0, len(memories))
	for index, memory := range memories {
		refs = append(refs, ExistingMemoryRef{Ref: fmt.Sprintf("M%d", index+1), Memory: memory})
	}
	return refs
}

// classifyExtractionError 把错误归类成非敏感的短错误码，供退避与排错使用，不落原始错误文本。
func classifyExtractionError(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, ErrMemoryInvalidOperation),
		errors.Is(err, ErrMemoryInvalidReference),
		errors.Is(err, ErrMemorySourceForbidden),
		errors.Is(err, ErrMemoryEvidenceMismatch):
		return "invalid_output"
	case errors.Is(err, ErrMemoryVersionConflict):
		return "version_conflict"
	default:
		return "error"
	}
}

// withDefaults 把非法/缺省的配置项回退到安全默认值，避免误配成 0 导致无 Worker、无租约或死循环。
func (c MemoryPipelineConfig) withDefaults() MemoryPipelineConfig {
	if c.WorkerCount <= 0 {
		c.WorkerCount = 2
	}
	if c.QueueSize <= 0 {
		c.QueueSize = 256
	}
	if c.ScanInterval <= 0 {
		c.ScanInterval = time.Minute
	}
	if c.LeaseDuration <= 0 {
		c.LeaseDuration = 90 * time.Second
	}
	if c.ExtractTimeout <= 0 {
		c.ExtractTimeout = 30 * time.Second
	}
	if c.TaskTimeout <= 0 {
		c.TaskTimeout = 60 * time.Second
	}
	if c.ScanBatchSize <= 0 {
		c.ScanBatchSize = 50
	}
	if c.MaxBatchMessages <= 0 {
		c.MaxBatchMessages = 20
	}
	if c.MaxBatchChars <= 0 {
		c.MaxBatchChars = 8000
	}
	if c.MemoryInputLimit <= 0 {
		c.MemoryInputLimit = 50
	}
	if c.MemoryInputChars <= 0 {
		c.MemoryInputChars = 4000
	}
	if c.BaseRetryBackoff <= 0 {
		c.BaseRetryBackoff = 5 * time.Second
	}
	if c.MaxRetryBackoff <= 0 {
		c.MaxRetryBackoff = 10 * time.Minute
	}
	if c.ShutdownGrace <= 0 {
		c.ShutdownGrace = 10 * time.Second
	}
	if c.ExtractorModel == "" {
		c.ExtractorModel = "deepseek/deepseek-v4-flash"
	}
	if c.ExtractorVersion == "" {
		c.ExtractorVersion = "memory-extractor-v1"
	}
	return c
}

func (c MemoryPipelineConfig) validate() error {
	if c.ExtractTimeout >= c.TaskTimeout {
		return errors.New("记忆抽取配置非法: extract_timeout 必须小于 task_timeout")
	}
	if c.LeaseDuration <= c.TaskTimeout+leaseCommitMargin {
		return fmt.Errorf("记忆抽取配置非法: lease_duration 必须大于 task_timeout + %s 提交余量", leaseCommitMargin)
	}
	if c.BaseRetryBackoff > c.MaxRetryBackoff {
		return errors.New("记忆抽取配置非法: retry_base_delay 不能大于 retry_max_delay")
	}
	if c.MaxBatchMessages < 2 {
		return errors.New("记忆抽取配置非法: max_batch_messages 至少为 2（一个完整 turn）")
	}
	return nil
}
