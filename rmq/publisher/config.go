package publisher

import (
	"fmt"
	"time"

	"github.com/kelchy/go-lib/log"
	"github.com/streadway/amqp"
)

// Config is the configuration for publisher.
type Config struct {
	// Name: name of the publisher queue.
	Name string `json:"name" mapstructure:"name"`
	// Exchange: exchange to publish to.
	Exchange string `json:"exchange" mapstructure:"exchange"`
	// Mandatory: if true, return an unroutable message with a Return method.
	Mandatory bool `json:"mandatory" mapstructure:"mandatory"`
	// Immediate: if true, request a delivery confirmation from the server.
	Immediate bool `json:"immediate" mapstructure:"immediate"`
	// AutoGenerateMessageID: if true, generate a message id for the message.
	AutoGenerateMessageID bool `json:"auto_generate_message_id" mapstructure:"auto_generate_message_id"`
	// PublisherConfirmed: if true, wait for publisher confirmation.
	PublisherConfirmed bool `json:"publisher_confirmed" mapstructure:"publisher_confirmed"`
	// Timeout: timeout for waiting for publisher confirmation.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
	// NoWait: if true, do not wait for the server to confirm the message.
	NoWait bool `json:"no_wait" mapstructure:"no_wait"`
}

// DefaultConfig is the default configuration for publisher.
func DefaultConfig(name string, exchange string) Config {
	return Config{
		Name:                  name,
		Exchange:              exchange,
		Mandatory:             false,
		Immediate:             false,
		AutoGenerateMessageID: true,
		PublisherConfirmed:    false,
		Timeout:               5 * time.Second,
		NoWait:                false,
	}
}

// ConnectionConfig is the configuration for connection.
type ConnectionConfig struct {
	// ConnURIs: list of connection URIs.
	ConnURIs []string `json:"conn_uris" mapstructure:"conn_uris"`
	// ReconnectInterval: interval between reconnect attempts.
	ReconnectInterval time.Duration `json:"reconnect_interval" mapstructure:"reconnect_interval"`
	// ReconnectMaxAttempt: max number of reconnect attempts.
	ReconnectMaxAttempt int `json:"reconnect_max_attempt" mapstructure:"reconnect_max_attempt"`
}

// DefaultConnectionConfig is the default configuration for connection.
func DefaultConnectionConfig(connURIs []string) ConnectionConfig {
	return ConnectionConfig{
		ConnURIs:            connURIs,
		ReconnectInterval:   5 * time.Second,
		ReconnectMaxAttempt: 3,
	}
}

// DefaultPublishMessage is the function to publish a default message.
func DefaultPublishMessage(message []byte) amqp.Publishing {
	return amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		Body:         message,
	}
}

// DefaultLogger is the default logger for the package
func DefaultLogger() ILogger {
	logger, err := log.New("standard")
	if err != nil {
		fmt.Println("failed to create logger: ", err)
	}
	return logger
}
