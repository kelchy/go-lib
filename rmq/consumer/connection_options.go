package rabbitmq

import (
	"time"

	"github.com/kelchy/go-lib/rmq/consumer/internal/logger"
)

const defaultReconnectInterval = time.Second * 5

// ConnectionOptions are used to describe how a new consumer will be created.
type ConnectionOptions struct {
	ReconnectInterval time.Duration
	Logger            logger.Logger
	Config            Config
}

// getDefaultConnectionOptions describes the options that will be used when a value isn't provided
func getDefaultConnectionOptions() ConnectionOptions {
	return ConnectionOptions{
		ReconnectInterval: defaultReconnectInterval,
		Logger:            logger.DefaultLogger,
		Config:            Config{},
	}
}

// WithConnectionOptionsReconnectInterval sets the reconnection interval
func WithConnectionOptionsReconnectInterval(interval time.Duration) func(options *ConnectionOptions) {
	return func(options *ConnectionOptions) {
		options.ReconnectInterval = interval
	}
}

// WithConnectionOptionsLogging sets logging to true on the consumer options
// and sets the
func WithConnectionOptionsLogging(options *ConnectionOptions) {
	options.Logger = logger.DefaultLogger
}

// WithConnectionOptionsLogger sets logging to true on the consumer options
// and sets the
func WithConnectionOptionsLogger(log logger.Logger) func(options *ConnectionOptions) {
	return func(options *ConnectionOptions) {
		options.Logger = log
	}
}

// WithConnectionOptionsConfig sets the Config used in the connection
func WithConnectionOptionsConfig(cfg Config) func(options *ConnectionOptions) {
	return func(options *ConnectionOptions) {
		options.Config = cfg
	}
}
