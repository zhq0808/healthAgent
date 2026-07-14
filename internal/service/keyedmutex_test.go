package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestKeyedMutexSerializesSameKey(t *testing.T) {
	locks := NewKeyedMutex()
	var active int32
	var maxActive int32
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := locks.Lock(context.Background(), "same")
			if err != nil {
				t.Errorf("lock same key: %v", err)
				return
			}
			defer release()
			current := atomic.AddInt32(&active, 1)
			for {
				observed := atomic.LoadInt32(&maxActive)
				if current <= observed || atomic.CompareAndSwapInt32(&maxActive, observed, current) {
					break
				}
			}
			atomic.AddInt32(&active, -1)
		}()
	}
	wg.Wait()

	if maxActive != 1 {
		t.Fatalf("max concurrent holders for same key = %d, want 1", maxActive)
	}
	if size := locks.size(); size != 0 {
		t.Fatalf("keyed mutex retained %d entries, want 0 after release", size)
	}
}

func TestKeyedMutexAllowsDifferentKeysConcurrently(t *testing.T) {
	locks := NewKeyedMutex()
	release1, err := locks.Lock(context.Background(), "a")
	if err != nil {
		t.Fatalf("lock a: %v", err)
	}
	release2, err := locks.Lock(context.Background(), "b") // 不同 key 不应阻塞
	if err != nil {
		t.Fatalf("lock b: %v", err)
	}
	if size := locks.size(); size != 2 {
		t.Fatalf("active keys = %d, want 2", size)
	}
	release1()
	release2()
	if size := locks.size(); size != 0 {
		t.Fatalf("keyed mutex retained %d entries, want 0", size)
	}
}

func TestKeyedMutexStopsWaitingWhenContextCanceled(t *testing.T) {
	locks := NewKeyedMutex()
	release, err := locks.Lock(context.Background(), "same")
	if err != nil {
		t.Fatalf("lock holder: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := locks.Lock(ctx, "same"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiting lock err=%v, want context deadline exceeded", err)
	}
	release()
	if size := locks.size(); size != 0 {
		t.Fatalf("keyed mutex retained %d entries after canceled waiter", size)
	}
}
