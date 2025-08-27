package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	rabbitmq "github.com/kelchy/go-lib/rmq/publisher"
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

	publisher, err := rabbitmq.NewPublisher(
		conn,
		rabbitmq.WithPublisherOptionsLogging,
		rabbitmq.WithPublisherOptionsExchangeName("events"),
		rabbitmq.WithPublisherOptionsExchangeDurable,
		rabbitmq.WithPublisherOptionsExchangeDeclare,
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer publisher.Close()

	publisher.NotifyReturn(func(r rabbitmq.Return) {
		// To handle returned messages
		fmt.Println("err message returned from server: ", string(r.MessageId), (r.Body))
	})

	publisher.NotifyPublish(func(c rabbitmq.Confirmation) {
		fmt.Println("message confirmed from server. ack: ", c.Ack)
	})

	publisher2, err := rabbitmq.NewPublisher(
		conn,
		rabbitmq.WithPublisherOptionsLogging,
		rabbitmq.WithPublisherOptionsExchangeName("events"),
		rabbitmq.WithPublisherOptionsExchangeDurable,
		rabbitmq.WithPublisherOptionsExchangeDeclare,
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer publisher2.Close()

	publisher2.NotifyReturn(func(r rabbitmq.Return) {
		// To handle returned messages
		fmt.Println("err message returned from server: ", string(r.MessageId), string(r.Body))
	})

	publisher2.NotifyPublish(func(c rabbitmq.Confirmation) {
		fmt.Println("message confirmed from server. ack: ", c.Ack)
	})

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

	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			err = publisher.PublishWithContext(
				context.Background(),
				[]byte("hello, world"),
				[]string{"test_routing_key"},
				rabbitmq.WithPublishOptionsContentType("application/json"),
				rabbitmq.WithPublishOptionsMandatory,
				rabbitmq.WithPublishOptionsPersistentDelivery,
				rabbitmq.WithPublishOptionsExchange("events"),
				rabbitmq.WithPublishOptionsAutoMessageID(),
			)
			if err != nil {
				fmt.Println(err)
			}
			err = publisher2.PublishWithContext(
				context.Background(),
				[]byte("hello, world 2"),
				[]string{"test_routing_key"},
				rabbitmq.WithPublishOptionsContentType("application/json"),
				rabbitmq.WithPublishOptionsMandatory,
				rabbitmq.WithPublishOptionsPersistentDelivery,
				rabbitmq.WithPublishOptionsExchange("events"),
				rabbitmq.WithPublishOptionsAutoMessageID(),
			)
			if err != nil {
				fmt.Println(err)
			}
		case <-done:
			fmt.Println("stopping publisher")
			return
		}
	}
}
