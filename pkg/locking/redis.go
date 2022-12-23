package locking

import (
	"context"
	"fmt"
	"github.com/bsm/redislock"
	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

type RedisDistributedLocker struct {
	client *redislock.Client
	redis  *redis.Client
}

func (rdl *RedisDistributedLocker) Obtain(ctx context.Context, req ctrl.Request) LockSession {
	lock, err := rdl.client.Obtain(ctx, createLockingId(req), time.Second*60, &redislock.Options{})
	if err == redislock.ErrNotObtained {
		return LockSession{lock, errors.New(ErrAlreadyLocked)}
	}
	return LockSession{lock, err}
}

func (rdl *RedisDistributedLocker) Done(ctx context.Context, session LockSession) {
	lock := session.id.(redislock.Lock)
	_ = lock.Release(ctx)
}

func (rdl *RedisDistributedLocker) Close() {
	_ = rdl.redis.Close()
}

func NewRedisDistributedLocker(proto string, host string, port int) *RedisDistributedLocker {
	r := redis.NewClient(&redis.Options{
		Network: proto,
		Addr:    fmt.Sprintf("%s:%v", host, port),
	})
	return &RedisDistributedLocker{
		redislock.New(r),
		r,
	}
}
