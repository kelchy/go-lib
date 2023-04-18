package rabbitmq

import (
	"errors"
	"fmt"
	"sync"

	"github.com/kelchy/go-lib/rmq/consumer/internal/channelmanager"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Action is an action that occurs after processed this delivery
type Action int

// Handler defines the handler of each Delivery and return Action
type Handler func(d Delivery) (action Action)

// EventHandler defines the handler of each Delivery and returns error if any
type EventHandler func(d Delivery) (err error)

const (
	// Ack default ack this msg after you have successfully processed this delivery.
	Ack Action = iota
	// NackDiscard the message will be dropped or delivered to a server configured dead-letter queue.
	NackDiscard
	// NackRequeue deliver this message to a different consumer.
	NackRequeue
	// Manual means message acknowledgement is left to the user using the msg.Ack() method
	Manual
)

// Consumer allows you to create and connect to queues for data consumption.
type Consumer struct {
	chanManager                *channelmanager.ChannelManager
	reconnectErrCh             <-chan error
	closeConnectionToManagerCh chan<- struct{}
	options                    ConsumerOptions

	isClosedMux *sync.RWMutex
	isClosed    bool
}

// Delivery captures the fields for a previously delivered message resident in
// a queue to be delivered by the server to a consumer from Channel.Consume or
// Channel.Get.
type Delivery struct {
	amqp.Delivery
}

// NewConsumer returns a new Consumer connected to the given rabbitmq server
// it also starts consuming on the given connection with automatic reconnection handling
// Do not reuse the returned consumer for anything other than to close it
func NewConsumer(
	conn *Conn,
	handler EventHandler,
	deadMsgHandler EventHandler,
	queue string,
	optionFuncs ...func(*ConsumerOptions),
) (*Consumer, error) {
	defaultOptions := getDefaultConsumerOptions(queue)
	options := &defaultOptions
	for _, optionFunc := range optionFuncs {
		optionFunc(options)
	}

	if conn.connectionManager == nil {
		return nil, errors.New("connection manager can't be nil")
	}

	chanManager, err := channelmanager.NewChannelManager(conn.connectionManager, options.Logger, conn.connectionManager.ReconnectInterval)
	if err != nil {
		return nil, err
	}
	reconnectErrCh, closeCh := chanManager.NotifyReconnect()

	consumer := &Consumer{
		chanManager:                chanManager,
		reconnectErrCh:             reconnectErrCh,
		closeConnectionToManagerCh: closeCh,
		options:                    *options,
		isClosedMux:                &sync.RWMutex{},
		isClosed:                   false,
	}

	err = consumer.startGoroutines(
		handler,
		deadMsgHandler,
		*options,
	)
	if err != nil {
		return nil, err
	}

	go func() {
		for err := range consumer.reconnectErrCh {
			consumer.options.Logger.Out("OK_CONSUME_RECONNECT", fmt.Sprintf("successful consumer recovery from: %v", err))
			err = consumer.startGoroutines(
				handler,
				deadMsgHandler,
				*options,
			)
			if err != nil {
				consumer.options.Logger.Error("ERR_CONSUMER_RECONNECT", fmt.Errorf("consumer closing, error restarting consumer goroutines after cancel or close: %v", err))
				return
			}
		}
	}()

	return consumer, nil
}

// Close cleans up resources and closes the consumer.
// It does not close the connection manager, just the subscription
// to the connection manager and the consuming goroutines.
// Only call once.
func (consumer *Consumer) Close() {
	consumer.isClosedMux.Lock()
	defer consumer.isClosedMux.Unlock()
	consumer.isClosed = true
	// close the channel so that rabbitmq server knows that the
	// consumer has been stopped.
	err := consumer.chanManager.Close()
	if err != nil {
		consumer.options.Logger.Error("WARN_CONSUMER_CLOSE-CHANNEL", fmt.Errorf("error while closing the channel: %v", err))
	}

	consumer.options.Logger.Out("OK_CONSUMER_CLOSE", "closing consumer")
	go func() {
		consumer.closeConnectionToManagerCh <- struct{}{}
	}()
}

// startGoroutines declares the queue if it doesn't exist,
// binds the queue to the routing key(s), and starts the goroutines
// that will consume from the queue
func (consumer *Consumer) startGoroutines(
	handler EventHandler,
	deadMsgHandler EventHandler,
	options ConsumerOptions,
) error {
	err := consumer.chanManager.QosSafe(
		options.QOSPrefetch,
		0,
		options.QOSGlobal,
	)
	if err != nil {
		return fmt.Errorf("declare qos failed: %w", err)
	}
	err = declareExchange(consumer.chanManager, options.ExchangeOptions)
	if err != nil {
		return fmt.Errorf("declare exchange failed: %w", err)
	}
	err = declareQueue(consumer.chanManager, options.QueueOptions)
	if err != nil {
		return fmt.Errorf("declare queue failed: %w", err)
	}
	err = declareBindings(consumer.chanManager, options)
	if err != nil {
		return fmt.Errorf("declare bindings failed: %w", err)
	}

	msgs, err := consumer.chanManager.ConsumeSafe(
		options.QueueOptions.Name,
		options.RabbitConsumerOptions.Name,
		options.RabbitConsumerOptions.AutoAck,
		options.RabbitConsumerOptions.Exclusive,
		false, // no-local is not supported by RabbitMQ
		options.RabbitConsumerOptions.NoWait,
		tableToAMQPTable(options.RabbitConsumerOptions.Args),
	)
	if err != nil {
		return err
	}

	for i := 0; i < options.Concurrency; i++ {
		go handlerGoroutine(consumer, msgs, options, handler, deadMsgHandler)
	}
	consumer.options.Logger.Out("OK_CONSUMER_GOROUTINE-HANDLER", fmt.Sprintf("Processing messages on %v goroutines", options.Concurrency))
	return nil
}

func (consumer *Consumer) getIsClosed() bool {
	consumer.isClosedMux.RLock()
	defer consumer.isClosedMux.RUnlock()
	return consumer.isClosed
}

func handlerGoroutine(consumer *Consumer, msgs <-chan amqp.Delivery, consumeOptions ConsumerOptions, handler EventHandler, deadMsgHandler EventHandler) {
	for msg := range msgs {
		if consumer.getIsClosed() {
			break
		}

		// Ack the message before processing if autoAck is enabled, if not the user will have to ack the message manually.
		if consumeOptions.RabbitConsumerOptions.AutoAck {
			err := msg.Ack(false)
			if err != nil {
				consumer.options.Logger.Error("ERR_CONSUMER_AUTO-ACK", fmt.Errorf("can't ack message: %v", err))
			}
		}

		// Check if message has been redelivered. If headers are present, it means the message has been redelivered at least once.
		// We pass the msg to the deadMsgHandler if the retry limit has been exceeded.
		headers := msg.Headers
		// This is only for instant retry on quorum queues (x-delivery-count)
		if deliveryCount, ok := headers["x-delivery-count"].(int64); ok {
			if int(deliveryCount) > consumeOptions.RabbitConsumerOptions.RetryLimit {
				if err := deadMsgHandler(Delivery{msg}); err != nil {
					consumer.options.Logger.Error("ERR_CONSUMER_DEAD-MESSAGE-HANDLER", fmt.Errorf("error in dead message handler: %v", err))
				}
				continue
			}
		}
		// This is for dlx retry (x-death)
		if xDeathContent, ok := headers["x-death"].([]interface{}); ok {
			var retryCount int64
			for _, content := range xDeathContent {
				table, _ := content.(amqp.Table)
				retryCount = table["count"].(int64)
				break
			}
			if int(retryCount) > consumeOptions.RabbitConsumerOptions.RetryLimit {
				if err := deadMsgHandler(Delivery{msg}); err != nil {
					consumer.options.Logger.Error("ERR_CONSUMER_DEAD-MESSAGE-HANDLER", fmt.Errorf("error in dead message handler: %v", err))
				}
				continue
			}
		}

		// Attempt to handle message, Ack should be performed by user in handler, we only handle error cases.
		if err := handler(Delivery{msg}); err != nil {
			consumer.options.Logger.Error("ERR_CONSUMER_HANDLER", fmt.Errorf("error in handler: %v", err))
			// Two options here, requeue directly into queue or requeue via dead letter exchange
			if consumeOptions.RabbitConsumerOptions.DlxRetry {
				err := msg.Nack(false, false)
				if err != nil {
					consumer.options.Logger.Error("ERR_CONSUMER_NACK-REQUEUE-DLX", fmt.Errorf("can't nack message: %v", err))
				}
			} else if consumeOptions.RabbitConsumerOptions.Retry && consumer.options.QueueOptions.Args["x-queue-type"] == "quorum" {
				err := msg.Nack(false, true)
				if err != nil {
					consumer.options.Logger.Error("ERR_CONSUMER_NACK-REQUEUE", fmt.Errorf("can't nack message: %v", err))
				}
			} else {
				if err := deadMsgHandler(Delivery{msg}); err != nil {
					consumer.options.Logger.Error("ERR_CONSUMER_DEAD-MESSAGE-HANDLER", fmt.Errorf("error in dead message handler: %v", err))
				}
			}
			continue
		}
	}
	consumer.options.Logger.Out("OK_CONSUMER_GOROUTINE-CLOSED", "rabbit consumer goroutine closed")
}
