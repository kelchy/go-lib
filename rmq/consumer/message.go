package consumer

import (
	"github.com/streadway/amqp"
)

// IMessage interface contains methods implemented by the package
type IMessage interface {
	GetID() string
	Ack(flag bool, opts ...option) error
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
func (m *Message) Ack(flag bool, opts ...option) error {
	options := newOptions(opts...)
	multiple, _ := options.Context.Value(multipleKey{}).(bool)
	requeue, _ := options.Context.Value(requeueKey{}).(bool)
	if flag {
		err := m.delivery.Ack(multiple)
		if err != nil {
			return err
		}
	} else {
		err := m.delivery.Reject(requeue)
		if err != nil {
			return err
		}
	}
	return nil
}

// Body returns the message body
func (m *Message) Body() []byte {
	return m.delivery.Body
}

// Headers returns the message headers
func (m *Message) Headers() map[string]interface{} {
	return m.delivery.Headers
}
