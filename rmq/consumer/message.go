package consumer

import (
	"github.com/streadway/amqp"
)

// IMessage interface contains methods implemented by the package
type IMessage interface {
	GetID() string
	Ack(multiple bool) error
	Nack(multiple bool, requeue bool) error
	Reject(requeue bool) error
	Headers() map[string]interface{}
	Body() []byte
}

// Message struct
type Message struct {
	delivery amqp.Delivery
}

// NewMessage returns a new Message
func NewMessage(d amqp.Delivery) IMessage {
	return &Message{
		delivery: d,
	}
}

// GetID returns the message id
func (m *Message) GetID() string {
	return m.delivery.MessageId
}

// Ack acknowledges the message
func (m *Message) Ack(multiple bool) error {
	return m.delivery.Ack(multiple)
}

// Nack nacks the message with the option to Nack multiple messages
func (m *Message) Nack(multiple bool, requeue bool) error {
	return m.delivery.Nack(multiple, requeue)
}

// Reject rejects the message
func (m *Message) Reject(requeue bool) error {
	return m.delivery.Reject(requeue)
}

// Body returns the message body
func (m *Message) Body() []byte {
	return m.delivery.Body
}

// Headers returns the message headers
func (m *Message) Headers() map[string]interface{} {
	return m.delivery.Headers
}
