package consumer

import (
	"fmt"
	"time"

	"github.com/kelchy/go-lib/log"
)

// ConnectionConfig is the configuration for connection creation.
type ConnectionConfig struct {
	// ConnURIs: list of connection URIs.
	ConnURIs []string `json:"conn_uris" mapstructure:"conn_uris"`
	// ReconnectInterval: interval between reconnect attempts.
	ReconnectInterval time.Duration `json:"reconnect_interval" mapstructure:"reconnect_interval"`
	// ReconnectMaxAttempt: max number of reconnect attempts.
	ReconnectMaxAttempt int `json:"reconnect_max_attempt" mapstructure:"reconnect_max_attempt"`
}

// DefaultConnectionConfig returns a default connection configuration.
func DefaultConnectionConfig(connURIs []string) ConnectionConfig {
	return ConnectionConfig{
		ConnURIs:            connURIs,
		ReconnectInterval:   5 * time.Second,
		ReconnectMaxAttempt: 3,
	}
}

// Config is the configuration for consumer creation.
type Config struct {
	Enabled   bool                   `json:"enabled" mapstructure:"enabled"`
	Name      string                 `json:"name" mapstructure:"name"`
	AutoAck   bool                   `json:"auto_ack" mapstructure:"auto_ack"`
	Exclusive bool                   `json:"exclusive" mapstructure:"exclusive"`
	NoLocal   bool                   `json:"no_local" mapstructure:"no_local"`
	NoWait    bool                   `json:"no_wait" mapstructure:"no_wait"`
	Args      map[string]interface{} `json:"args" mapstructure:"args"`

	// Fair dispatch
	EnabledPrefetch bool `json:"enabled_prefetch" mapstructure:"enabled_prefetch"`
	PrefetchCount   int  `json:"prefetch_count" mapstructure:"prefetch_count"`
	PrefetchSize    int  `json:"prefetch_size" mapstructure:"prefetch_size"`
	Global          bool `json:"global" mapstructure:"global"`
}

// DefaultConfig returns a default consumer configuration.
func DefaultConfig(name string) Config {
	return Config{
		Enabled:         true,
		Name:            name,
		AutoAck:         true,
		Exclusive:       false,
		NoLocal:         false,
		NoWait:          false,
		Args:            nil,
		EnabledPrefetch: true,
		PrefetchCount:   1,
		PrefetchSize:    0,
		Global:          false,
	}
}

// QueueConfig is the configuration for queue creation.
type QueueConfig struct {
	Name string `json:"name" mapstructure:"name"`
	// Durable: if true, the queue will survive broker restarts.
	Durable bool `json:"durable" mapstructure:"durable"`
	// AutoDelete: if true, the queue will be deleted when the last consumer unsubscribes.
	AutoDelete bool `json:"auto_delete" mapstructure:"auto_delete"`
	// Exclusive: if true, only accessible by the connection that declares it.
	Exclusive bool `json:"exclusive" mapstructure:"exclusive"`
	// NoWait: if true, the server will not respond to the method.
	NoWait bool `json:"no_wait" mapstructure:"no_wait"`
	// Additional Arguments
	Args map[string]interface{} `json:"args" mapstructure:"args"`
}

// DefaultQueueConfig returns a default queue configuration.
func DefaultQueueConfig(name string) QueueConfig {
	return QueueConfig{
		Name:       name,
		Durable:    true,
		AutoDelete: true,
		Exclusive:  false,
		NoWait:     false,
		Args:       nil,
	}
}

// QueueBindConfig is the configuration for queue binding to an exchange with a routing key.
type QueueBindConfig struct {
	Queue    string `json:"queue" mapstructure:"queue"`
	Exchange string `json:"exchange" mapstructure:"exchange"`
	// Key which the queue possesses. Routing Key (on message) will be compared to the Binding Key (on route) to decide if message is to be routed
	BindingKey string                 `json:"binding_key" mapstructure:"binding_key"`
	NoWait     bool                   `json:"no_wait" mapstructure:"no_wait"`
	Args       map[string]interface{} `json:"args" mapstructure:"args"`
}

// DefaultQueueBindConfig returns a default queue binding configuration.
func DefaultQueueBindConfig(exchange string, queue string, bindingKey string) QueueBindConfig {
	return QueueBindConfig{
		Queue:      queue,
		Exchange:   exchange,
		BindingKey: bindingKey,
		NoWait:     false,
		Args:       nil,
	}
}

// DefaultLogger generates a logger to be used with the consumer.
func DefaultLogger() ILogger {
	logger, err := log.New("standard")
	if err != nil {
		fmt.Println("failed to create logger: ", err)
	}
	return logger
}

// MessageRetryConfig is the configuration for message retry.
type MessageRetryConfig struct {
	// retry
	Enabled           bool `json:"enabled" mapstructure:"enabled"`
	HandleDeadMessage bool `json:"handle_dead_message" mapstructure:"handle_dead_message"`
	RetryCountLimit   int  `json:"retry_count_limit" mapstructure:"retry_count_limit"`
}

// DefaultMessageRetryConfig returns a default message retry configuration.
func DefaultMessageRetryConfig() MessageRetryConfig {
	return MessageRetryConfig{
		Enabled:           true,
		HandleDeadMessage: true,
		RetryCountLimit:   2,
	}
}
