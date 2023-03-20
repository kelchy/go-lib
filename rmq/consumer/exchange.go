package consumer

import "github.com/streadway/amqp"

// NewExchange creates a new exchange or ensures an exchange exists that matches the provided configuration
func NewExchange(amqpChannel *amqp.Channel, exchangeConfig ExchangeConfig) error {
	err := amqpChannel.ExchangeDeclare(exchangeConfig.Name, exchangeConfig.Kind, exchangeConfig.Durable, exchangeConfig.AutoDelete, exchangeConfig.Internal, exchangeConfig.NoWait, exchangeConfig.Args)
	return err
}
