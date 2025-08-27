package redis

import (
	"context"
	"fmt"
	"time"

	redis "github.com/go-redis/redis/v8"
)

// Get - implementation of redis GET, ctx can be nil
func (r Client) Get(ctx context.Context, key string) (string, error) {
	val, err := r.Client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		r.log.Error("REDIS_GET", err)
		return val, err
	}
	return val, nil
}

// Keys - implementation of redis KEYS, ctx can be nil
func (r Client) Keys(ctx context.Context, match string) ([]string, error) {
	var list []string
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = r.Client.Scan(ctx, cursor, match, 0).Result()
		if err != nil {
			r.log.Error("REDIS_KEYS", err)
			return list, err
		}
		list = append(list, keys...)
		if cursor == 0 {
			break
		}
	}
	return list, nil
}

// TTL - implementation of redis TTL, ctx can be nil
// -- returns in time.Duration which is always in nanoseconds; convert to seconds
// -- it returns -1 (-1ns) when there is no associated expiry
// -- it returns -2 (-2ns) when there is no key
func (r Client) TTL(ctx context.Context, key string) (int64, error) {
	val, err := r.Client.TTL(ctx, key).Result()

	// return a negative number to not confuse it with a positive value response
	if err != nil {
		r.log.Error("REDIS_TTL", err)
		return -3, err
	}

	if val == -2 {
		errMsg := fmt.Errorf("no TTL found for key")
		r.log.Error("REDIS_TTL_NOT_FOUND", errMsg)
		return -2, errMsg
	}

	if val == -1 {
		errMsg := fmt.Errorf("TTL does not expire")
		r.log.Error("REDIS_TTL_NO_EXPIRY", errMsg)
		return -1, errMsg
	}

	return int64(val / time.Second), nil
}
