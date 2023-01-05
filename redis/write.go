package redis

import (
	"time"
	"errors"
	"context"
	redis "github.com/go-redis/redis/v8"
)

var lockPrefix = "lock_"

// Lock - implementation of redis distributed lock, ctx can be nil
func (r Client) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	tmpkey := lockPrefix + key
	args := redis.SetArgs{
		TTL:	ttl,
		Mode:	"NX",
	}
	s := r.Client.SetArgs(ctx, tmpkey, "", args)
	res, err := s.Result()

	// if key is locked, returns redis.Nil
	// implied that res != OK
	if err == redis.Nil {
		return false, nil
	}
	if res == "OK" {
		return true, nil
	}
	if err != nil {
		r.log.Error("REDIS_LOCK", err)
		return false, err
	}
	// by default don't give lock
	return false, nil
}

// Unlock - implementation of redis distributed unlock, ctx can be nil
func (r Client) Unlock(ctx context.Context, key string) (bool, error) {
	tmpkey := lockPrefix + key
	res, err := r.Del(ctx, tmpkey)
	if err != nil {
		err = errors.New("Unlock: failed")
		r.log.Error("REDIS_UNLOCK", err)
		return false, err
	}
	resbool := false
	if res > 0 {
		resbool = true
	}
	return resbool, err
}

// Set - implementation of redis SET, ctx can be nil
func (r Client) Set(ctx context.Context, key string, value string, ttl time.Duration) (string, error) {
	res, err := r.Client.Set(ctx, key, value, ttl).Result()
	if err != nil {
		r.log.Error("REDIS_SET", err)
		return res, err
	}
	return res, nil
}

// SetNX - implementation of redis SET with NX flag, ctx can be nil
func (r Client) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	res, err := r.Client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		r.log.Error("REDIS_SETNX", err)
		return res, err
	}
	return res, nil
}

// Del - implementation of redis DEL, ctx can be nil
func (r Client) Del(ctx context.Context, key string) (int64, error) {
	res, err := r.Client.Del(ctx, key).Result()
	if err != nil {
		r.log.Error("REDIS_DEL", err)
		return res, err
	}
	return res, nil
}
