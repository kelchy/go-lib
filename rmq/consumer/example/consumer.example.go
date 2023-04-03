package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kelchy/go-lib/rmq/consumer"
)

func main() {
	eventProcessor := &eventProcessor{}
	err := consumer.New(
		consumer.DefaultConnectionConfig([]string{os.Getenv("RMQ_URI")}),
		// Queue names should be the same in QueueConfig and QueueBindConfig
		consumer.DefaultQueueConfig("test-queue-logging"),
		consumer.DefaultQueueBindConfig("test-exchange", "test-queue-logging", "test-routing-key"),
		consumer.DefaultConfig("test-consumer"),
		consumer.DefaultMessageRetryConfig(),
		eventProcessor,
		consumer.DefaultLogger())
	if err != nil {
		fmt.Println("failed to create consumer: ", err)
	}

	// If you want to verify the presence of queue and exchanges
	// Not required if consumer is init using New() as above
	conn, _ := consumer.NewConnection(consumer.DefaultConnectionConfig([]string{os.Getenv("RMQ_URI")}), consumer.DefaultLogger())
	connChan, _ := conn.Channel()
	exDeclareErr := consumer.NewExchange(connChan, consumer.DefaultExchangeConfig("test-exchange", "direct"))
	_, queueDeclareErr := consumer.NewQueue(connChan, consumer.QueueConfig{
		Name:       "test-queue-logging",
		Durable:    true,
		AutoDelete: true,
		Exclusive:  false,
		NoWait:     false,
		Args:       nil,
	})
	qBindErr := consumer.NewQueueBind(connChan, "test-exchange", "test-queue-logging", "test-routing-key", false, nil)
	if exDeclareErr != nil {
		fmt.Println("failed to declare exchange: ", exDeclareErr)
		return
	}
	if queueDeclareErr != nil {
		fmt.Println("failed to declare queue: ", queueDeclareErr)
		return
	}
	if qBindErr != nil {
		fmt.Println("failed to bind queue: ", qBindErr)
		return
	}
	fmt.Println("queue and exchange verified")
	// Leave the consumer running for 30 seconds before exiting, only for example purposes
	time.Sleep(30 * time.Second)
}

// EventProcessor is an example of a consumer event processor.
type eventProcessor struct{}

func (*eventProcessor) ProcessEvent(ctx context.Context, message consumer.IMessage) error {
	fmt.Printf("Recieved message: ID: %s, Message: %s\n", message.GetID(), string(message.Body()))
	return nil
}

func (*eventProcessor) ProcessDeadMessage(ctx context.Context, message consumer.IMessage, err error) error {
	fmt.Printf("Recieved dead message: ID: %s, Message: %s, Error: %v", message.GetID(), string(message.Body()), err)
	return nil
}
