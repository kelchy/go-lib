package consumer

import "github.com/streadway/amqp"

// NewQueue declares a new queue or ensures a queue exists that matches the provided configuration
func NewQueue(amqpChannel *amqp.Channel, queueConfig QueueConfig) (amqp.Queue, error) {
	q, err := amqpChannel.QueueDeclare(queueConfig.Name, queueConfig.Durable, queueConfig.AutoDelete, queueConfig.Exclusive, queueConfig.NoWait, queueConfig.Args)
	return q, err
}
