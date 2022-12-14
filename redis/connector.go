package redis

import (
	"context"
	"errors"
	redis "github.com/go-redis/redis/v8"
	"github.com/kelchy/go-lib/log"
)

// KeepTTL - helper to expose KeepTTL
var KeepTTL = redis.KeepTTL

// Client - instance created when initializing client
type Client struct {
	Client	*redis.Client
	ctx	context.Context
	log	log.Log
}

// New - constructor to create an instance of the client
func New(uri string) (Client, error) {
	l, _ := log.New("")
	var r Client
	opt, err := redis.ParseURL(uri)
	if err != nil {
		l.Error("REDIS_PARSE_URL", err)
		return r, err
	}
	r.Client = redis.NewClient(opt)
	r.ctx = context.Background()
	r.log = l
	pong, err := r.Client.Ping(r.ctx).Result()
	if err != nil {
		l.Error("REDIS_PING", err)
		return r, err
	} else if pong != "PONG" {
		err = errors.New("redis ping no pong")
		l.Error("REDIS_PONG", err)
		return r, err
	}
	return r, nil
}
