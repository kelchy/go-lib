package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	rabbitmq "github.com/kelchy/go-lib/rmq/consumer"
)

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

	// Verify that exchange exists (Not needed if it is declared in NewConsumer)
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

	// Verify that queue exists (Not needed if it is declared in NewConsumer)
	if err := rabbitmq.DeclareQueue(
		conn,
		rabbitmq.QueueOptions{
			Name:       "my_queue",
			Declare:    true,
			Durable:    true,
			AutoDelete: false,
			Exclusive:  false,
			NoWait:     false,
			Args: map[string]interface{}{
				"x-dead-letter-exchange":    "events",
				"x-dead-letter-routing-key": "test_routing_key_dlk",
			},
		},
	); err != nil {
		fmt.Println(err)
		return
	}

	// Verify that queue is bound to exchange (Not needed binding is declared in NewConsumer)
	if err := rabbitmq.DeclareBinding(
		conn,
		rabbitmq.BindingDeclareOptions{
			QueueName:    "my_queue",
			ExchangeName: "events",
			RoutingKey:   "test_routing_key",
			NoWait:       false,
			Args:         nil,
			Declare:      true,
		}); err != nil {
		fmt.Println(err)
		return
	}

	if err := rabbitmq.DeclareQueue(
		conn,
		rabbitmq.QueueOptions{
			Name:       "my_queue_dlx",
			Declare:    true,
			Durable:    true,
			AutoDelete: false,
			Exclusive:  false,
			NoWait:     false,
			Args: map[string]interface{}{
				"x-dead-letter-exchange":    "events",
				"x-dead-letter-routing-key": "test_routing_key",
				"x-message-ttl":             10000,
			},
		},
	); err != nil {
		fmt.Println(err)
		return
	}

	// Verify that queue is bound to exchange (Not needed binding is declared in NewConsumer)
	if err := rabbitmq.DeclareBinding(
		conn,
		rabbitmq.BindingDeclareOptions{
			QueueName:    "my_queue_dlx",
			ExchangeName: "events",
			RoutingKey:   "test_routing_key_dlk",
			NoWait:       false,
			Args:         nil,
			Declare:      true,
		}); err != nil {
		fmt.Println(err)
		return
	}

	consumer, err := rabbitmq.NewConsumer(
		conn,
		eventHandler,
		deadMessageHandler,
		"my_queue",
		rabbitmq.WithConsumerOptionsConcurrency(2),
		rabbitmq.WithConsumerOptionsRoutingKey("test_routing_key"),
		rabbitmq.WithConsumerOptionsExchangeName("events"),
		rabbitmq.WithConsumerOptionsQueueArgs(map[string]interface{}{
			"x-dead-letter-exchange":    "events",
			"x-dead-letter-routing-key": "test_routing_key_dlk",
		}),
		rabbitmq.WithConsumerOptionsConsumerRetryLimit(3),
		rabbitmq.WithConsumerOptionsConsumerDlxRetry,
		rabbitmq.WithConsumerOptionsQueueDurable,
	)
	if err != nil {
		fmt.Println(err)
	}
	defer consumer.Close()

	consumer2, err := rabbitmq.NewConsumer(
		conn,
		eventHandler,
		deadMessageHandler,
		"my_queue",
		rabbitmq.WithConsumerOptionsConcurrency(2),
		rabbitmq.WithConsumerOptionsRoutingKey("test_routing_key"),
		rabbitmq.WithConsumerOptionsExchangeName("events"),
		rabbitmq.WithConsumerOptionsQueueArgs(map[string]interface{}{
			"x-dead-letter-exchange":    "events",
			"x-dead-letter-routing-key": "test_routing_key_dlk",
		}),
		rabbitmq.WithConsumerOptionsConsumerRetryLimit(3),
		rabbitmq.WithConsumerOptionsConsumerDlxRetry,
		rabbitmq.WithConsumerOptionsQueueDurable,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer2.Close()

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
	return fmt.Errorf("TEST_ERR")
	err := d.Ack(false)
	if err != nil {
		return err
	}
	return nil
}

func deadMessageHandler(d rabbitmq.Delivery) error {
	fmt.Printf("Dead Message Received: %s, %v \n", string(d.MessageId), string(d.Body))
	err := d.Ack(false)
	if err != nil {
		return err
	}
	return nil
}
