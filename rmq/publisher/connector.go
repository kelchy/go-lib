package publisher

import (
	"context"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

// Publisher is the publisher struct
type Publisher struct {
	pubChan      *amqp.Channel
	pubConfig    Config
	conn         *amqp.Connection
	connPool     IConnectionPool
	errorConn    chan *amqp.Error
	errorPubChan chan *amqp.Error
	logger       ILogger
}

// ILogger is the interface for logger
type ILogger interface {
	Debug(key string, message string)
	Out(key string, message string)
	Error(key string, err error)
}

// New creates a new publisher
func New(connConfig ConnectionConfig, pubConfig Config, logger ILogger) (*Publisher, error) {
	publisher := Publisher{
		logger:    logger,
		pubConfig: pubConfig,
	}
	publisher.connPool = newConnectionPool(connConfig.ConnURIs...)
	publisher.connect(connConfig)
	go publisher.listenOnChanClose()
	return &publisher, nil
}

// Publish publishes a message to the queue
// Returns message id and error
func (p *Publisher) Publish(ctx context.Context, routingKey string, toPublish amqp.Publishing) (string, error) {
	// Create a publishing object to be sent to RMQ
	if p.pubConfig.AutoGenerateMessageID {
		uuid, err := NewUUID()
		if err != nil {
			return "", fmt.Errorf("failed to auto generate message id: %v", err)
		}
		toPublish.MessageId = uuid
	}
	// Publish the message
	pubErr := p.pubChan.Publish(p.pubConfig.Exchange, routingKey, p.pubConfig.Mandatory, p.pubConfig.Immediate, toPublish)
	if pubErr != nil {
		p.logger.Error("ERR_RMQ-PUBLISHER_FAIL-PUBLISH", pubErr)
	}
	return toPublish.MessageId, nil
}

func (p *Publisher) connect(connConfig ConnectionConfig) error {
	attempts := 0
	for attempts <= connConfig.ReconnectMaxAttempt {
		p.logger.Out("RMQ-PUBLISHER", "Connecting to RabbitMQ")
		// Make a connection to RMQ
		conn, err := p.connPool.GetCon()
		if err != nil {
			p.logger.Error("ERR_RMQ-PUBLISHER_FAIL-CONNECT", err)
			time.Sleep(connConfig.ReconnectInterval)
			// Wait before retrying
			continue
		}
		p.conn = conn
		p.errorConn = make(chan *amqp.Error)
		p.conn.NotifyClose(p.errorConn)

		// Open a channel for publishing
		pubChan, pubChanErr := p.openChannel()
		if pubChanErr != nil {
			p.logger.Error("ERR_RMQ-PUBLISHER_FAIL-OPEN-CHANNEL", pubChanErr)
			return pubChanErr
		}
		p.pubChan = pubChan
		p.errorPubChan = make(chan *amqp.Error)
		p.pubChan.NotifyClose(p.errorPubChan)
		p.logger.Out("RMQ-PUBLISHER", "Connected to RabbitMQ")
		return nil
	}
	return nil
}

func (p *Publisher) openChannel() (*amqp.Channel, error) {
	if p.conn == nil || p.conn.IsClosed() {
		return nil, fmt.Errorf("connection is not open")
	}
	return p.conn.Channel()
}

func (p *Publisher) listenOnChanClose() {
	for {
		select {
		case err := <-p.errorPubChan:
			if err != nil {
				p.logger.Error("ERR_RMQ-PUBLISHER_FAIL-CHANNEL-CLOSE", err)
				if !p.conn.IsClosed() {
					errClose := p.conn.Close()
					if errClose != nil {
						p.logger.Error("ERR_RMQ-PUBLISHER_FAIL-CHANNEL-CLOSE", errClose)
					}
				}
			}
		}
	}
}
