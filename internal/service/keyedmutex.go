package service

import (
	"context"
	"sync"
)

// KeyedMutex 提供“按 key 串行、不同 key 并行”的互斥能力，并在某个 key 无人使用/等待后回收它，
// 避免长期运行时 key 数量随用户增长造成内存泄漏。
//
// 用途：单实例内保证同一 user_id 的多个 Session 抽取串行，不并发修改同一份记忆。
type KeyedMutex struct {
	mu    sync.Mutex
	locks map[string]*keyedEntry
}

type keyedEntry struct {
	gate chan struct{}
	refs int // 正在持有或等待该 key 的调用方数量，降到 0 时从表中删除
}

// NewKeyedMutex 构造一个空的 KeyedMutex。
func NewKeyedMutex() *KeyedMutex {
	return &KeyedMutex{locks: make(map[string]*keyedEntry)}
}

// Lock 获取 key 对应的锁。等待期间 ctx 取消时立即退出，避免任务超时后仍无限排队。
// 成功返回的释放函数可重复调用但只会实际释放一次（通常 defer）。
func (k *KeyedMutex) Lock(ctx context.Context, key string) (func(), error) {
	k.mu.Lock()
	entry := k.locks[key]
	if entry == nil {
		entry = &keyedEntry{gate: make(chan struct{}, 1)}
		entry.gate <- struct{}{}
		k.locks[key] = entry
	}
	entry.refs++
	k.mu.Unlock()

	select {
	case <-entry.gate:
		var once sync.Once
		return func() {
			once.Do(func() {
				entry.gate <- struct{}{}
				k.releaseRef(key, entry)
			})
		}, nil
	case <-ctx.Done():
		k.releaseRef(key, entry)
		return nil, ctx.Err()
	}
}

func (k *KeyedMutex) releaseRef(key string, entry *keyedEntry) {
	k.mu.Lock()
	defer k.mu.Unlock()
	entry.refs--
	if entry.refs == 0 && k.locks[key] == entry {
		delete(k.locks, key)
	}
}

// size 返回当前活跃 key 数量，供测试断言 key 回收。
func (k *KeyedMutex) size() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.locks)
}
