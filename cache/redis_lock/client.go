package redis_lock

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"time"
)

var (
	ErrFailedToLock = errors.New(`some one else already lock`)
)

type Client struct {
	client redis.Cmdable
}

func (c *Client) TryLock(ctx context.Context,
	key string, expireSpan time.Duration) (*Lock, error) {
	val := uuid.New().String()
	ok, err := c.client.SetNX(ctx, key, val, expireSpan).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrFailedToLock
	}
	return newLock(c.client, key, val, expireSpan), nil
}

func (c *Client) Lock(ctx context.Context, retry RetryStrategy, timeout time.Duration, key string, expireSpan time.Duration) (*Lock, error) {
	val := uuid.New().String()
	var checker *time.Timer
	defer func() {
		if checker != nil {
			checker.Stop()
		}
	}()
	for {
		ctx2, cancel := context.WithTimeout(ctx, timeout)
		res, err := c.client.Eval(ctx2, luaLock, []string{key}, val, expireSpan.Seconds()).Result()
		cancel()
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if res == "OK" {
			return newLock(c.client, key, val, expireSpan), nil
		}
		interval, ok := retry.Next()
		if !ok {
			if err == nil {
				err = fmt.Errorf("others own lock: %w", ErrFailedToLock)
			}
			return nil, err
		}

		if checker == nil {
			checker = time.NewTimer(interval)
		} else {
			checker.Reset(interval)
		}
		select {
		case <-checker.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
