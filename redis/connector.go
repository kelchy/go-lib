package redis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	redis "github.com/go-redis/redis/v8"
	"github.com/kelchy/go-lib/log"
	"strings"
)

// KeepTTL - helper to expose KeepTTL
var KeepTTL = redis.KeepTTL

// Client - instance created when initializing client
type Client struct {
	Client *redis.Client
	ctx    context.Context
	log    log.Log
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

	// hack for redis labs CA issue
	if strings.Contains(uri, "rediss") && strings.Contains(opt.Addr, "redislabs.com") {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(CACert))
		opt.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:      caCertPool,
		}
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

// NewSecure - constructor to create an instance of the client
func NewSecure(uri string, clientCertPath string, clientKeyPath string, skipVerify bool) (Client, error) {
	l, _ := log.New("")
	var r Client

	tlsCert, tlsErr := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if tlsErr != nil {
		l.Error("err", tlsErr)
		return r, tlsErr
	}

	opt, err := redis.ParseURL(uri)
	if err != nil {
		l.Error("REDIS_PARSE_URL", err)
		return r, err
	}

	// tlsCerts generated with tls.X509KeyPair()
	// enable TLS connections if input provided
	if tlsCert.Certificate != nil {
		opt.TLSConfig = &tls.Config{
			Certificates:       []tls.Certificate{tlsCert},
			InsecureSkipVerify: skipVerify,
		}
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
