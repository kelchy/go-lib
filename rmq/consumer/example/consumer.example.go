package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	rabbitmq "github.com/kelchy/go-lib/rmq/consumer"
)

/**
 * This example demonstrates how to create a consumer with retry and dead letter queue
 * The consumer will consume messages from the queue "q_events" with routing key "events.basic"
 * If the message is Nacked, it will be sent to the dead letter queue "q_event_retry"
 * The dead letter queue will retry the message 3 times with a delay of 10 seconds between each retry
 * If the message is Nacked after the 3rd retry, it will be handled by the dead message handler
 */

func main() {
	conn, err := rabbitmq.NewConn(
		os.Getenv("RMQ_URI"),
		rabbitmq.WithConnectionOptionsLogging,
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	// Verify that exchange exists
	if err := rabbitmq.DeclareExchange(
		conn,
		rabbitmq.ExchangeOptions{
			Name:       "events",
			Kind:       "direct",
			Declare:    true,
			Durable:    true,
			AutoDelete: false,
			Internal:   false,
			NoWait:     false,
			Args:       nil,
		},
	); err != nil {
		fmt.Println(err)
		return
	}

	// Consumer options can be changed according to your needs
	rabbitmq.NewConsumer(
		conn,
		eventHandler,
		deadMessageHandler,
		"q_events",
		rabbitmq.WithConsumerOptionsExchangeName("events"),
		rabbitmq.WithConsumerOptionsRoutingKey("events.basic"),
		rabbitmq.WithConsumerOptionsConsumerDlxRetry,
		rabbitmq.WithConsumerOptionsConsumerRetryLimit(3),
		rabbitmq.WithConsumerOptionsConsumerAutoAck(false),
		rabbitmq.WithConsumerOptionsQueueDurable,
		rabbitmq.WithConsumerOptionsQueueArgs(
			map[string]interface{}{
				"x-dead-letter-exchange":    "events",
				"x-dead-letter-routing-key": "retry.events.basic",
			},
		),
	)

	// Declares the DLQ and binds it back to the original queue
	rabbitmq.DeclareQueue(
		conn,
		rabbitmq.QueueOptions{
			Name:       "q_event_retry",
			Durable:    true,
			AutoDelete: false,
			Exclusive:  false,
			NoWait:     false,
			Args: map[string]interface{}{
				"x-message-ttl":             10000,
				"x-dead-letter-exchange":    "events",
				"x-dead-letter-routing-key": "events.basic",
			},
			Declare: true,
		},
	)
	rabbitmq.DeclareBinding(
		conn,
		rabbitmq.BindingDeclareOptions{
			QueueName:    "q_event_retry",
			ExchangeName: "events",
			RoutingKey:   "retry.events.basic",
		},
	)

	// block main thread - wait for shutdown signal
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	fmt.Println("awaiting signal")
	<-done
	fmt.Println("stopping consumer")
}

func eventHandler(d rabbitmq.Delivery) error {
	fmt.Printf("consumed: %s, %v \n", string(d.MessageId), string(d.Body))
	return fmt.Errorf("nack")
	if err := d.Ack(false); err != nil {
		return err
	}
	return nil
}

// Alternatively can use a wrapper for the event handler so you will not need to call ack manually within each different consumer
func eventHandlerWrapper(f rabbitmq.EventHandler) func(msg rabbitmq.Delivery) error {
	return func(msg rabbitmq.Delivery) error {
		if err := f(msg); err != nil {
			return err
		}
		if err := msg.Ack(false); err != nil {
			return err
		}
		return nil
	}
}

func deadMessageHandler(d rabbitmq.Delivery) error {
	fmt.Printf("Dead Message Received: %s, %v \n", string(d.MessageId), string(d.Body))
	if err := d.Ack(false); err != nil {
		return err
	}
	return nil
}
