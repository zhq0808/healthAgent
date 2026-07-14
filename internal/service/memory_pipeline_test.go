package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
)

// ---- 可控 fake 依赖 ----

type stubMemoryAccess struct {
	mu         sync.Mutex
	listResult []Memory
	listErr    error
	applyErr   error
	applied    []ApplyExtractionInput
	onApply    func(ApplyExtractionInput)
}

func (s *stubMemoryAccess) ListCurrentMemories(_ context.Context, _ string, _ MemoryBudget) ([]Memory, error) {
	return s.listResult, s.listErr
}

func (s *stubMemoryAccess) ApplyExtraction(_ context.Context, input ApplyExtractionInput) (ApplyExtractionResult, error) {
	s.mu.Lock()
	s.applied = append(s.applied, input)
	hook := s.onApply
	err := s.applyErr
	s.mu.Unlock()
	if hook != nil {
		hook(input)
	}
	if err != nil {
		return ApplyExtractionResult{}, err
	}
	return ApplyExtractionResult{ToResultSeq: input.ToResultSeq}, nil
}

func (s *stubMemoryAccess) appliedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.applied)
}

type stubFailure struct {
	sessionID string
	code      string
	ctxErr    error
}

type stubExtractionRepo struct {
	mu          sync.Mutex
	lookupUser  func(sessionID string) (string, bool, error)
	acquire     func(sessionID, userID string) (ExtractionLease, bool, error)
	loadBatch   func(sessionID, userID string, from, to int64) ([]ExtractionTurn, error)
	scanBacklog func() ([]string, error)
	recordErr   error
	failures    []stubFailure
	scanCalls   int
}

func (r *stubExtractionRepo) LookupSessionUser(_ context.Context, sessionID string) (string, bool, error) {
	if r.lookupUser != nil {
		return r.lookupUser(sessionID)
	}
	return "u1", true, nil
}

func (r *stubExtractionRepo) AcquireExtractionLease(_ context.Context, sessionID, userID string, _ time.Duration) (ExtractionLease, bool, error) {
	if r.acquire != nil {
		return r.acquire(sessionID, userID)
	}
	return ExtractionLease{SessionID: sessionID, UserID: userID, LeaseToken: "lt-" + sessionID, FromResultSeq: 0, ToResultSeq: 2}, true, nil
}

func (r *stubExtractionRepo) LoadExtractionBatch(_ context.Context, sessionID, userID string, from, to int64) ([]ExtractionTurn, error) {
	if r.loadBatch != nil {
		return r.loadBatch(sessionID, userID, from, to)
	}
	return []ExtractionTurn{{
		ResultSeq:        2,
		UserMessage:      BatchMessageRef{MessageID: "um-" + sessionID, Role: "user", Content: "hi", Seq: 1},
		AssistantMessage: BatchMessageRef{MessageID: "am-" + sessionID, Role: "assistant", Content: "ok", Seq: 2},
	}}, nil
}

func (r *stubExtractionRepo) RecordExtractionFailure(ctx context.Context, sessionID, _, _, errorCode string, _, _ time.Duration) error {
	r.mu.Lock()
	r.failures = append(r.failures, stubFailure{sessionID: sessionID, code: errorCode, ctxErr: ctx.Err()})
	r.mu.Unlock()
	return r.recordErr
}

func (r *stubExtractionRepo) ScanExtractionBacklog(_ context.Context, _ int) ([]string, error) {
	r.mu.Lock()
	r.scanCalls++
	r.mu.Unlock()
	if r.scanBacklog != nil {
		return r.scanBacklog()
	}
	return nil, nil
}

func (r *stubExtractionRepo) recordedFailures() []stubFailure {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]stubFailure(nil), r.failures...)
}

type stubExtractor struct {
	extract func(ctx context.Context, input ExtractionInput) ([]LLMMemoryOperation, error)
}

func (s stubExtractor) Extract(ctx context.Context, input ExtractionInput) ([]LLMMemoryOperation, error) {
	if s.extract != nil {
		return s.extract(ctx, input)
	}
	return nil, nil
}

func testPipeline(access MemoryAccess, repo ExtractionRepository, extractor MemoryExtractor) *MemoryPipeline {
	pipeline, err := NewMemoryPipeline(access, repo, extractor, MemoryPipelineConfig{
		WorkerCount:      2,
		QueueSize:        8,
		ScanInterval:     time.Hour,
		LeaseDuration:    time.Minute,
		ExtractTimeout:   time.Second,
		TaskTimeout:      2 * time.Second,
		ScanBatchSize:    10,
		MaxBatchMessages: 20,
		MaxBatchChars:    10000,
		MemoryInputLimit: 50,
		MemoryInputChars: 4000,
		BaseRetryBackoff: time.Second,
		MaxRetryBackoff:  time.Minute,
		ShutdownGrace:    2 * time.Second,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		panic(err)
	}
	return pipeline
}

// ---- process() 直接测试（无 goroutine，确定性）----

func TestPipelineProcessAppliesWindowAndRefs(t *testing.T) {
	access := &stubMemoryAccess{}
	repo := &stubExtractionRepo{}
	extractor := stubExtractor{extract: func(_ context.Context, input ExtractionInput) ([]LLMMemoryOperation, error) {
		if len(input.BatchMessages) != 2 || input.BatchMessages[0].Ref != "N1" || input.BatchMessages[1].Ref != "N2" {
			t.Errorf("batch refs = %+v, want N1/N2", input.BatchMessages)
		}
		return nil, nil
	}}
	pipeline := testPipeline(access, repo, extractor)

	pipeline.process("s1")

	if access.appliedCount() != 1 {
		t.Fatalf("apply calls = %d, want 1", access.appliedCount())
	}
	applied := access.applied[0]
	if applied.FromResultSeq != 0 || applied.ToResultSeq != 2 || applied.LeaseToken != "lt-s1" {
		t.Fatalf("applied window = %+v, want from 0 to 2 lease lt-s1", applied)
	}
	if got := pipeline.Metrics().Processed; got != 1 {
		t.Fatalf("processed metric = %d, want 1", got)
	}
}

func TestPipelineProcessSkipsWhenNotAcquired(t *testing.T) {
	access := &stubMemoryAccess{}
	repo := &stubExtractionRepo{acquire: func(_, _ string) (ExtractionLease, bool, error) {
		return ExtractionLease{}, false, nil // 已被他人持有 / 无积压 / 退避中
	}}
	pipeline := testPipeline(access, repo, stubExtractor{})

	pipeline.process("s1")

	if access.appliedCount() != 0 {
		t.Fatalf("apply calls = %d, want 0 when lease not acquired", access.appliedCount())
	}
	if got := pipeline.Metrics().Skipped; got != 1 {
		t.Fatalf("skipped metric = %d, want 1", got)
	}
}

func TestPipelineProcessSkipsWhenSessionMissing(t *testing.T) {
	access := &stubMemoryAccess{}
	repo := &stubExtractionRepo{lookupUser: func(string) (string, bool, error) { return "", false, nil }}
	pipeline := testPipeline(access, repo, stubExtractor{})

	pipeline.process("gone")

	if access.appliedCount() != 0 {
		t.Fatalf("apply calls = %d, want 0 when session missing", access.appliedCount())
	}
}

func TestPipelineProcessRecordsFailureOnExtractorTimeout(t *testing.T) {
	access := &stubMemoryAccess{}
	repo := &stubExtractionRepo{}
	extractor := stubExtractor{extract: func(context.Context, ExtractionInput) ([]LLMMemoryOperation, error) {
		return nil, context.DeadlineExceeded
	}}
	pipeline := testPipeline(access, repo, extractor)

	pipeline.process("s1")

	if access.appliedCount() != 0 {
		t.Fatalf("apply calls = %d, want 0 when extractor times out", access.appliedCount())
	}
	failures := repo.recordedFailures()
	if len(failures) != 1 || failures[0].code != "timeout" {
		t.Fatalf("failures = %+v, want single timeout", failures)
	}
	if got := pipeline.Metrics().Failed; got != 1 {
		t.Fatalf("failed metric = %d, want 1", got)
	}
}

func TestPipelineProcessRecordsFailureOnApplyError(t *testing.T) {
	access := &stubMemoryAccess{applyErr: errors.New("db down")}
	repo := &stubExtractionRepo{}
	pipeline := testPipeline(access, repo, stubExtractor{})

	pipeline.process("s1")

	failures := repo.recordedFailures()
	if len(failures) != 1 || failures[0].code != "error" {
		t.Fatalf("failures = %+v, want single error", failures)
	}
}

func TestPipelineProcessDoesNotRecordFailureOnCursorConflict(t *testing.T) {
	access := &stubMemoryAccess{applyErr: ErrExtractionCursorConflict}
	repo := &stubExtractionRepo{}
	pipeline := testPipeline(access, repo, stubExtractor{})

	pipeline.process("s1")

	if failures := repo.recordedFailures(); len(failures) != 0 {
		t.Fatalf("failures = %+v, want none on lease takeover", failures)
	}
	if got := pipeline.Metrics().Skipped; got != 1 {
		t.Fatalf("skipped metric = %d, want 1", got)
	}
}

func TestPipelineProcessCapsBatchByLimit(t *testing.T) {
	access := &stubMemoryAccess{}
	repo := &stubExtractionRepo{
		acquire: func(sessionID, userID string) (ExtractionLease, bool, error) {
			return ExtractionLease{SessionID: sessionID, UserID: userID, LeaseToken: "lt", FromResultSeq: 0, ToResultSeq: 6}, true, nil
		},
		loadBatch: func(string, string, int64, int64) ([]ExtractionTurn, error) {
			return []ExtractionTurn{
				{ResultSeq: 2, UserMessage: BatchMessageRef{MessageID: "u1", Role: "user", Content: "a", Seq: 1}, AssistantMessage: BatchMessageRef{MessageID: "a1", Role: "assistant", Content: "b", Seq: 2}},
				{ResultSeq: 4, UserMessage: BatchMessageRef{MessageID: "u2", Role: "user", Content: "c", Seq: 3}, AssistantMessage: BatchMessageRef{MessageID: "a2", Role: "assistant", Content: "d", Seq: 4}},
				{ResultSeq: 6, UserMessage: BatchMessageRef{MessageID: "u3", Role: "user", Content: "e", Seq: 5}, AssistantMessage: BatchMessageRef{MessageID: "a3", Role: "assistant", Content: "f", Seq: 6}},
			}, nil
		},
	}
	pipeline := testPipeline(access, repo, stubExtractor{})
	pipeline.cfg.MaxBatchMessages = 2 // 只能容纳一个 turn（2 条消息）

	pipeline.process("s1")

	applied := access.applied[0]
	if applied.ToResultSeq != 2 {
		t.Fatalf("effective to seq = %d, want 2 (only first turn fits)", applied.ToResultSeq)
	}
	if len(applied.BatchMessages) != 2 {
		t.Fatalf("batch messages = %d, want 2", len(applied.BatchMessages))
	}
}

func TestPipelineProcessTruncatesOversizedFirstTurnWithinHardLimit(t *testing.T) {
	access := &stubMemoryAccess{}
	repo := &stubExtractionRepo{loadBatch: func(string, string, int64, int64) ([]ExtractionTurn, error) {
		return []ExtractionTurn{{
			ResultSeq:        2,
			UserMessage:      BatchMessageRef{MessageID: "u1", Role: "user", Content: "用户消息很长", Seq: 1},
			AssistantMessage: BatchMessageRef{MessageID: "a1", Role: "assistant", Content: "助手消息也很长", Seq: 2},
		}}, nil
	}}
	var extracted ExtractionInput
	pipeline := testPipeline(access, repo, stubExtractor{extract: func(_ context.Context, input ExtractionInput) ([]LLMMemoryOperation, error) {
		extracted = input
		return nil, nil
	}})
	pipeline.cfg.MaxBatchChars = 5

	pipeline.process("s1")

	if len(extracted.BatchMessages) != 2 {
		t.Fatalf("batch messages=%d, want one complete turn", len(extracted.BatchMessages))
	}
	chars := utf8.RuneCountInString(extracted.BatchMessages[0].Content) + utf8.RuneCountInString(extracted.BatchMessages[1].Content)
	if chars != 5 || extracted.BatchMessages[0].Content != "用户消息很" {
		t.Fatalf("truncated batch chars=%d user=%q assistant=%q", chars, extracted.BatchMessages[0].Content, extracted.BatchMessages[1].Content)
	}
	if access.appliedCount() != 1 {
		t.Fatalf("apply calls=%d, want cursor progress after bounded extraction", access.appliedCount())
	}
}

func TestPipelineRecordFailureUsesFreshContext(t *testing.T) {
	repo := &stubExtractionRepo{}
	pipeline := testPipeline(&stubMemoryAccess{}, repo, stubExtractor{})
	pipeline.recordFailure(ExtractionLease{SessionID: "s1", UserID: "u1", LeaseToken: "lt"}, "timeout")

	failures := repo.recordedFailures()
	if len(failures) != 1 || failures[0].ctxErr != nil {
		t.Fatalf("failures=%+v, want active cleanup context", failures)
	}
}

// ---- Worker Pool / Notify / 补扫 / 关闭 ----

func TestPipelineNotifyProcessesSession(t *testing.T) {
	done := make(chan string, 1)
	access := &stubMemoryAccess{onApply: func(input ApplyExtractionInput) { done <- input.SessionID }}
	pipeline := testPipeline(access, &stubExtractionRepo{}, stubExtractor{})
	pipeline.Start()
	defer pipeline.Close()

	pipeline.Notify("s-notify")

	select {
	case sessionID := <-done:
		if sessionID != "s-notify" {
			t.Fatalf("processed session = %q, want s-notify", sessionID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("notify was not processed within timeout")
	}
}

func TestPipelineDuplicateNotificationsOnlyApplyOnce(t *testing.T) {
	done := make(chan struct{}, 1)
	var acquired atomic.Bool
	repo := &stubExtractionRepo{acquire: func(sessionID, userID string) (ExtractionLease, bool, error) {
		if !acquired.CompareAndSwap(false, true) {
			return ExtractionLease{}, false, nil
		}
		return ExtractionLease{SessionID: sessionID, UserID: userID, LeaseToken: "lt", ToResultSeq: 2}, true, nil
	}}
	access := &stubMemoryAccess{onApply: func(ApplyExtractionInput) { done <- struct{}{} }}
	pipeline := testPipeline(access, repo, stubExtractor{})
	pipeline.Start()
	defer pipeline.Close()

	pipeline.Notify("s1")
	pipeline.Notify("s1")
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("first notification was not applied")
	}
	time.Sleep(20 * time.Millisecond)
	if calls := access.appliedCount(); calls != 1 {
		t.Fatalf("apply calls=%d, want duplicate notification deduplicated by lease/cursor", calls)
	}
}

func TestPipelineScannerProcessesBacklog(t *testing.T) {
	done := make(chan string, 1)
	access := &stubMemoryAccess{onApply: func(input ApplyExtractionInput) { done <- input.SessionID }}
	var handedOut atomic.Bool
	repo := &stubExtractionRepo{scanBacklog: func() ([]string, error) {
		if handedOut.CompareAndSwap(false, true) {
			return []string{"s-backlog"}, nil // 只在第一次补扫返回积压，避免重复处理
		}
		return nil, nil
	}}
	pipeline := testPipeline(access, repo, stubExtractor{})
	pipeline.Start()
	defer pipeline.Close()

	select {
	case sessionID := <-done:
		if sessionID != "s-backlog" {
			t.Fatalf("backfilled session = %q, want s-backlog", sessionID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("startup backfill did not process backlog within timeout")
	}
}

func TestPipelineSerializesSameUserAcrossSessions(t *testing.T) {
	var active int32
	var maxActive int32
	var wg sync.WaitGroup
	wg.Add(2)
	extractor := stubExtractor{extract: func(context.Context, ExtractionInput) ([]LLMMemoryOperation, error) {
		current := atomic.AddInt32(&active, 1)
		for {
			observed := atomic.LoadInt32(&maxActive)
			if current <= observed || atomic.CompareAndSwapInt32(&maxActive, observed, current) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond) // 制造重叠窗口
		atomic.AddInt32(&active, -1)
		return nil, nil
	}}
	// 两个 Session 同一用户 u1。
	repo := &stubExtractionRepo{
		lookupUser: func(string) (string, bool, error) { return "u1", true, nil },
		acquire: func(sessionID, userID string) (ExtractionLease, bool, error) {
			return ExtractionLease{SessionID: sessionID, UserID: userID, LeaseToken: "lt-" + sessionID, ToResultSeq: 2}, true, nil
		},
	}
	access := &stubMemoryAccess{onApply: func(ApplyExtractionInput) { wg.Done() }}
	pipeline := testPipeline(access, repo, extractor)
	pipeline.Start()
	defer pipeline.Close()

	pipeline.Notify("s1")
	pipeline.Notify("s2")
	wg.Wait()

	if maxActive != 1 {
		t.Fatalf("max concurrent same-user extractions = %d, want 1 (keyed lock serializes)", maxActive)
	}
}

func TestPipelineNotifyDropsWhenQueueFull(t *testing.T) {
	// 不 Start：没有 Worker 消费，队列容量 1，第二次投递必然被丢弃并计数。
	pipeline, err := NewMemoryPipeline(&stubMemoryAccess{}, &stubExtractionRepo{}, stubExtractor{}, MemoryPipelineConfig{
		WorkerCount: 1, QueueSize: 1, ShutdownGrace: time.Second,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("new pipeline: %v", err)
	}

	pipeline.Notify("first")
	pipeline.Notify("second")

	if got := pipeline.Metrics().Dropped; got < 1 {
		t.Fatalf("dropped metric = %d, want >= 1 when queue full", got)
	}
}

func TestPipelineCloseIsGracefulAndDropsAfterClose(t *testing.T) {
	pipeline := testPipeline(&stubMemoryAccess{}, &stubExtractionRepo{}, stubExtractor{})
	pipeline.Start()

	if err := pipeline.Close(); err != nil {
		t.Fatalf("graceful close error = %v, want nil", err)
	}

	before := pipeline.Metrics().Dropped
	pipeline.Notify("after-close")
	if got := pipeline.Metrics().Dropped; got != before+1 {
		t.Fatalf("dropped after close = %d, want %d", got, before+1)
	}
	// 重复 Close 幂等。
	if err := pipeline.Close(); err != nil {
		t.Fatalf("second close error = %v, want nil", err)
	}
}

func TestPipelineCloseReturnsAfterGraceWhenExtractorIgnoresCancellation(t *testing.T) {
	entered := make(chan struct{})
	unblock := make(chan struct{})
	extractor := stubExtractor{extract: func(context.Context, ExtractionInput) ([]LLMMemoryOperation, error) {
		close(entered)
		<-unblock
		return nil, context.Canceled
	}}
	pipeline := testPipeline(&stubMemoryAccess{}, &stubExtractionRepo{}, extractor)
	pipeline.cfg.ShutdownGrace = 20 * time.Millisecond
	pipeline.Start()
	pipeline.Notify("s1")
	<-entered

	started := time.Now()
	err := pipeline.Close()
	elapsed := time.Since(started)
	close(unblock)
	pipeline.wg.Wait()
	if err == nil {
		t.Fatal("close error=nil, want bounded timeout error")
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("close blocked for %s, want bounded return", elapsed)
	}
}

func TestNewMemoryPipelineRejectsUnsafeConfig(t *testing.T) {
	base := MemoryPipelineConfig{
		WorkerCount: 1, QueueSize: 1, ScanInterval: time.Second,
		LeaseDuration: 20 * time.Second, ExtractTimeout: time.Second, TaskTimeout: 10 * time.Second,
		ScanBatchSize: 1, MaxBatchMessages: 2, MaxBatchChars: 100,
		MemoryInputLimit: 1, MemoryInputChars: 100,
		BaseRetryBackoff: time.Second, MaxRetryBackoff: time.Minute, ShutdownGrace: time.Second,
	}
	tests := []struct {
		name   string
		mutate func(*MemoryPipelineConfig)
	}{
		{name: "extract timeout not below task", mutate: func(c *MemoryPipelineConfig) { c.ExtractTimeout = c.TaskTimeout }},
		{name: "lease lacks commit margin", mutate: func(c *MemoryPipelineConfig) { c.LeaseDuration = c.TaskTimeout + leaseCommitMargin }},
		{name: "retry range reversed", mutate: func(c *MemoryPipelineConfig) { c.BaseRetryBackoff = 2 * time.Minute }},
		{name: "cannot fit one turn", mutate: func(c *MemoryPipelineConfig) { c.MaxBatchMessages = 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := base
			test.mutate(&cfg)
			if _, err := NewMemoryPipeline(&stubMemoryAccess{}, &stubExtractionRepo{}, stubExtractor{}, cfg, slog.Default()); err == nil {
				t.Fatal("new pipeline error=nil, want unsafe config rejected")
			}
		})
	}
}
