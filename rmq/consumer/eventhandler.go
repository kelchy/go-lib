package consumer

import (
	"context"
	"fmt"

	"github.com/streadway/amqp"
)

// IEventHandler interface contains methods implemented by the package
type IEventHandler interface {
	HandleEvent(context.Context, IMessage)
	Retry(context.Context, IMessage, error)
	HandleDeadMessage(context.Context, IMessage, error)
}

// IClientHandler interface contains methods to be implemented by the user of the package
type IClientHandler interface {
	ProcessEvent(context.Context, IMessage) error
	ProcessDeadMessage(context.Context, IMessage, error) error
}

// EventHandler struct
type EventHandler struct {
	processor   IClientHandler
	logger      ILogger
	retryConfig *MessageRetryConfig
}

// NewEventHandler returns a new EventHandler
func NewEventHandler(processor IClientHandler, logger ILogger, retryConfig *MessageRetryConfig) IEventHandler {
	return &EventHandler{
		processor:   processor,
		logger:      logger,
		retryConfig: retryConfig,
	}
}

// HandleEvent handles the event received from the queue
func (e *EventHandler) HandleEvent(ctx context.Context, message IMessage) {
	err := e.processor.ProcessEvent(ctx, message)
	if err != nil {
		// Attempts to retry the message if the retry is enabled
		if e.retryConfig.Enabled {
			e.Retry(ctx, message, err)
			return
		}
		e.HandleDeadMessage(ctx, message, err)
		return
	}
}

// Retry retries the message if the retry config is enabled
func (e *EventHandler) Retry(ctx context.Context, message IMessage, err error) {
	headers := message.Headers()
	if headers == nil { // in case of 1st retry no headers are present
		errAck := message.Nack(false, false)
		if errAck != nil {
			e.logger.Error(fmt.Sprintf("ERR_EVENT_HANDLER-FAIL-MSG-ACK-%s", message.GetID()), errAck)
		}
		return
	}
	if xDeathContent, ok := headers["x-death"].([]interface{}); ok {
		for _, content := range xDeathContent {
			table, _ := content.(amqp.Table)
			retryCount, _ := table["count"].(int64)
			if int(retryCount) < e.retryConfig.RetryCountLimit {
				errAck := message.Nack(false, false)
				if errAck != nil {
					e.logger.Error(fmt.Sprintf("ERR_EVENT_HANDLER-FAIL-MSG-ACK-%s", message.GetID()), errAck)
					return
				}
				return
			}
			e.HandleDeadMessage(ctx, message, fmt.Errorf("retry count %d exceeded for msg %s", int(retryCount), message.GetID()))
			// Golint flags this return but it is needed
			return
		}
	}
	e.HandleDeadMessage(ctx, message, err)
}

// HandleDeadMessage handles the dead message
func (e *EventHandler) HandleDeadMessage(ctx context.Context, message IMessage, err error) {
	if e.retryConfig.HandleDeadMessage {
		handleDMErr := e.processor.ProcessDeadMessage(ctx, message, err)
		if handleDMErr != nil {
			e.logger.Error(fmt.Sprintf("ERR_EVENT_HANDLER-DEAD-MSG-FAIL-MSG-ACK-%s", message.GetID()), handleDMErr)
		}
	}
}
