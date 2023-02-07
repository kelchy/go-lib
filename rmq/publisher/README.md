# RMQ Consumer

This library is a wrapper around RMQ functions to make interaction with RMQ simpler and safer

## Quickstart

Refer to the below code snippet for how to set up a publisher called `test-publisher` publishing events to an exchange `test-exchange` with a routing key `test-routing-key`. 
```
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kelchy/go-lib/rmq/publisher"
)

func main() {
	// If using the default publisher config, you can just pass your publisher name and exchange name
	p, err := publisher.New(
		publisher.DefaultConnectionConfig([]string{os.Getenv("RMQ_URI")}),
		publisher.DefaultConfig("test-publisher", "test-exchange"),
		publisher.DefaultLogger())
	if err != nil {
		// Publisher failed to create and connect
		panic(err)
	}

	// Publish a message
	// Message to be published should be in a amqp Publishing object
	// Can simply use the default message if you don't need to set any custom properties
	message := []byte("test message from publisher")
	messageId, pubErr := p.Publish(context.TODO(), "test-routing-key", publisher.DefaultPublishMessage(message))
	if pubErr != nil {
		// Failed to publish message
		panic(pubErr)
	}
	fmt.Println("Message published with id: ", messageId)
}
```