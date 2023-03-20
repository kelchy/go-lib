package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

// Consumer is the struct that contains the consumer configuration
type Consumer struct {
	consumerChan       *amqp.Channel
	consumerConfig     Config
	queueConfig        QueueConfig
	queueBindConfig    QueueBindConfig
	messageRetryConfig MessageRetryConfig
	conn               *amqp.Connection
	connPool           IConnectionPool
	errorConn          chan *amqp.Error
	errorConsumerChan  chan *amqp.Error
	handler            IEventHandler
	logger             ILogger
}

// ILogger is the interface for the logger required by the package
type ILogger interface {
	Debug(key string, message string)
	Out(key string, message string)
	Error(key string, err error)
}

// New creates a new consumer, should use this by default
func New(connConfig ConnectionConfig, queueConfig QueueConfig, queueBindConfig QueueBindConfig, consumerConfig Config, msgRetryConfig MessageRetryConfig, processor IClientHandler, logger ILogger) error {
	// Set up connection to RabbitMQ
	c := Consumer{
		queueConfig:        queueConfig,
		queueBindConfig:    queueBindConfig,
		consumerConfig:     consumerConfig,
		messageRetryConfig: msgRetryConfig,
		logger:             logger,
	}
	c.connPool = newConnectionPool(connConfig.ConnURIs...)
	c.connect(connConfig)
	go c.listenOnChanClose()

	// Creates a new queue if it does not exist
	conQueue, qDeclareErr := c.consumerChan.QueueDeclare(c.queueConfig.Name, c.queueConfig.Durable, c.queueConfig.AutoDelete, c.queueConfig.Exclusive, c.queueConfig.NoWait, c.queueConfig.Args)
	if qDeclareErr != nil {
		logger.Error("ERR_RMQ-CONSUMER_FAIL-DECLARE-QUEUE", qDeclareErr)
		return qDeclareErr
	}
	// Binds the queue to the exchange
	qBindErr := c.consumerChan.QueueBind(c.queueBindConfig.Queue, c.queueBindConfig.BindingKey, c.queueBindConfig.Exchange, c.queueBindConfig.NoWait, c.queueBindConfig.Args)
	if qBindErr != nil {
		logger.Error("ERR_RMQ-CONSUMER_FAIL-BIND-QUEUE", qBindErr)
		return qBindErr
	}
	// Creates a new event handler
	c.handler = NewEventHandler(processor, c.logger, &c.messageRetryConfig)
	// Creates a consumer on the queue
	if c.consumerConfig.EnabledPrefetch {
		err := c.consumerChan.Qos(c.consumerConfig.PrefetchCount, c.consumerConfig.PrefetchSize, c.consumerConfig.Global)
		if err != nil {
			c.logger.Error("ERR_CONSUMER-FAILED-PREFETCH-ADD", err)
			return err
		}
	}
	msgs, err := c.consumerChan.Consume(c.queueBindConfig.Queue, c.consumerConfig.Name, c.consumerConfig.AutoAck,
		c.consumerConfig.Exclusive, c.consumerConfig.NoLocal, c.consumerConfig.NoWait, c.consumerConfig.Args)
	if err != nil {
		c.logger.Error(fmt.Sprintf("ERR_CONSUMER-FAILED-CONSUMER-REGISTER-%s", c.consumerConfig.Name), err)
		return err
	}
	// Starts consuming messages
	go func(msgs <-chan amqp.Delivery) {
		for msg := range msgs {
			c.handler.HandleEvent(context.TODO(), NewMessage(msg))
		}
	}(msgs)
	c.logger.Out("RMQ-CONSUMER", fmt.Sprintf("Consumer %s is listening on queue %s", c.consumerConfig.Name, conQueue.Name))
	return nil
}

// NewConnection creates a new connection to RabbitMQ. For use when you want to initialize queues and exchanges or verify that they are present
func NewConnection(connConfig ConnectionConfig, logger ILogger) (*amqp.Connection, error) {
	attempts := 0
	for attempts <= connConfig.ReconnectMaxAttempt {
		logger.Out("RMQ-CONSUMER", "Connecting to RabbitMQ")
		// Make a connection to RMQ
		conn, err := amqp.Dial(connConfig.ConnURIs[0])
		if err != nil {
			logger.Error("ERR_RMQ-CONSUMER_FAIL-CONNECT", err)
			time.Sleep(connConfig.ReconnectInterval)
			// Wait before retrying
			continue
		}
		logger.Out("RMQ-CONSUMER", "Connected to RabbitMQ")
		return conn, nil
	}
	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d attempts", connConfig.ReconnectMaxAttempt)
}

// OpenChannel opens a new channel on the connection
func OpenChannel(conn *amqp.Connection, logger ILogger) (*amqp.Channel, error) {
	amqpchan, err := conn.Channel()
	if err != nil {
		logger.Error("ERR_RMQ-CONSUMER_FAIL-OPEN-CHANNEL", err)
		return nil, err
	}
	return amqpchan, nil
}

func (c *Consumer) connect(connConfig ConnectionConfig) error {
	attempts := 0
	for attempts <= connConfig.ReconnectMaxAttempt {
		c.logger.Out("RMQ-CONSUMER", "Connecting to RabbitMQ")
		// Make a connection to RMQ
		conn, err := c.connPool.getCon()
		if err != nil {
			c.logger.Error("ERR_RMQ-CONSUMER_FAIL-CONNECT", err)
			time.Sleep(connConfig.ReconnectInterval)
			// Wait before retrying
			continue
		}
		c.conn = conn
		c.errorConn = make(chan *amqp.Error)
		c.conn.NotifyClose(c.errorConn)

		// Open a channel for publishing
		consumerChan, conChanErr := c.openChannel()
		if conChanErr != nil {
			c.logger.Error("ERR_RMQ-CONSUMER_FAIL-OPEN-CHANNEL", conChanErr)
			return conChanErr
		}
		c.consumerChan = consumerChan
		c.errorConsumerChan = make(chan *amqp.Error)
		c.consumerChan.NotifyClose(c.errorConsumerChan)
		c.logger.Out("RMQ-CONSUMER", "Connected to RabbitMQ")
		return nil
	}
	return nil
}

func (c *Consumer) openChannel() (*amqp.Channel, error) {
	if c.conn == nil || c.conn.IsClosed() {
		return nil, fmt.Errorf("connection is not open")
	}
	return c.conn.Channel()
}

func (c *Consumer) listenOnChanClose() {
	for {
		select {
		case err := <-c.errorConsumerChan:
			if err != nil {
				c.logger.Error("ERR_RMQ-PUBLISHER_FAIL-CHANNEL-CLOSE", err)
				if !c.conn.IsClosed() {
					errClose := c.conn.Close()
					if errClose != nil {
						c.logger.Error("ERR_RMQ-PUBLISHER_FAIL-CHANNEL-CLOSE", errClose)
					}
				}
			}
		}
	}
}
