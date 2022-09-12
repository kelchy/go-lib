package redis

import (
	"context"
	redis "github.com/go-redis/redis/v8"
)

// Get - implementation of redis GET
func (r Client) Get(ctx context.Context, key string) (string, error) {
	val, err := r.Client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		r.log.Error("REDIS_GET", err)
		return val, err
	}
	return val, nil
}

// Keys - implementation of redis KEYS
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
