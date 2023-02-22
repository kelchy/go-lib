# RMQ Consumer

This library is a wrapper around RMQ functions to make interaction with RMQ simpler and safer

## Quickstart

Refer to the below code snippet for how to set up a consumer called `test-consumer` consuming events from a queue `test-queue` bound to an exchange `test-exchange` on a binding key `test-routing-key`. 
```
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kelchy/go-lib/rmq/consumer"
)

func main() {
	eventProcessor := &EventProcessor{}
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

	// Leave the consumer running for 30 seconds before exiting, only for example purposes
	time.Sleep(30 * time.Second)
}

// Declare what to do when messages come in on the queue here
type EventProcessor struct{}

func (*EventProcessor) ProcessEvent(ctx context.Context, message consumer.IMessage) error {
	fmt.Printf("Recieved message: ID: %s, Message: %s\n", message.GetID(), string(message.Body()))
	// Fill in what to do with the consumed messages here
    return nil
}

func (*EventProcessor) ProcessDeadMessage(ctx context.Context, message consumer.IMessage, err error) error {
	fmt.Printf("Recieved dead message: ID: %s, Message: %s, Error: %v", message.GetID(), string(message.Body()), err)
    // Fill in what to do with the dead messages here
	return nil
}

```