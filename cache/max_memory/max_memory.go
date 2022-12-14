//go:build !answer

package cache

import (
	"context"
	"github.com/gotomicro/ekit/list"
	"sync"
	"time"
)

type MaxMemoryCache struct {
	Cache
	max  int64
	used int64

	list *list.LinkedList[string]

	mutex *sync.Mutex
}

func NewMaxMemoryCache(max int64, cache Cache) *MaxMemoryCache {
	m := &MaxMemoryCache{
		Cache: cache,
		max:   max,
		list:  list.NewLinkedList[string](),
		mutex: &sync.Mutex{},
	}
	m.Cache.OnEvicted(m.evicted)
	return m
}

func (m *MaxMemoryCache) Set(ctx context.Context, key string, val []byte, expiration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, err := m.Cache.LoadAndDelete(ctx, key); err != nil {
		return err
	}
	for m.used+int64(len(val)) > m.max {
		k, err := m.list.Get(0)
		if err != nil {
			return err
		}
		if err := m.Cache.Delete(ctx, k); err != nil {

			return err
		}

	}
	if err := m.Cache.Set(ctx, key, val, expiration); err != nil {
		return err
	}
	m.used = m.used + int64(len(val))
	if err := m.list.Append(key); err != nil {
		return err
	}
	return nil
}

func (m *MaxMemoryCache) Get(ctx context.Context, key string) (val []byte, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if val, err = m.Cache.Get(ctx, key); err == nil {
		m.deleteKey(key)
		_ = m.list.Append(key)
	}
	return val, err
}

func (m *MaxMemoryCache) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.Cache.Delete(ctx, key)
}

func (m *MaxMemoryCache) LoadAndDelete(ctx context.Context, key string) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.Cache.LoadAndDelete(ctx, key)
}

func (m *MaxMemoryCache) OnEvicted(f func(key string, val []byte)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Cache.OnEvicted(func(key string, val []byte) {
		m.evicted(key, val)
		f(key, val)
	})
}

func (m *MaxMemoryCache) evicted(key string, val []byte) {
	m.used = m.used - int64(len(val))
	m.deleteKey(key)
}

func (m *MaxMemoryCache) deleteKey(key string) {
	for i := 0; i < m.list.Len(); i++ {
		ele, _ := m.list.Get(i)
		if ele == key {
			_, _ = m.list.Delete(i)
			return
		}
	}
}
