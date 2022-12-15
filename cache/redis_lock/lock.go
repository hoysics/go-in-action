package redis_lock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v9"
	"time"
)

var (
	luaUnlock  string
	luaRefresh string
	luaLock    string
)

var (
	ErrNotHoldLock = errors.New("rlock: 未持有锁")
)

type Lock struct {
	client     redis.Cmdable
	key        string
	value      string
	expireSpan time.Duration
}

func newLock(client redis.Cmdable, key string, value string, expireSpan time.Duration) *Lock {
	return &Lock{
		client:     client,
		key:        key,
		value:      value,
		expireSpan: expireSpan,
	}
}

func (l *Lock) Unlock(ctx context.Context) error {
	res, err := l.client.Eval(ctx, luaUnlock, []string{l.key}, l.value).Int64()
	if err == redis.Nil {
		return ErrNotHoldLock
	}
	if err != nil {
		return err
	}
	if res != 1 {
		return ErrNotHoldLock
	}
	return nil
}

func (l *Lock) Refresh(ctx context.Context) error {
	res, err := l.client.Eval(ctx, luaRefresh, []string{l.key}, l.value, l.expireSpan.Seconds()).Int64()
	if err != nil {
		return err
	}
	if res != 1 {
		return ErrNotHoldLock
	}
	return nil
}
