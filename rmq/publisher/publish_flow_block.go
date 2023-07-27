package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (publisher *Publisher) startNotifyFlowHandler() {
	notifyFlowChan := publisher.chanManager.NotifyFlowSafe(make(chan bool))
	publisher.disablePublishDueToFlowMux.Lock()
	publisher.disablePublishDueToFlow = false
	publisher.disablePublishDueToFlowMux.Unlock()

	for ok := range notifyFlowChan {
		publisher.disablePublishDueToFlowMux.Lock()
		if ok {
			publisher.options.Logger.Error("WARN_RMQ-PUBLISH-FLOW_START-NOTIFY-FLOW-HANDLER", fmt.Errorf("pausing publishing due to flow request from server"))
			publisher.disablePublishDueToFlow = true
		} else {
			publisher.disablePublishDueToFlow = false
			publisher.options.Logger.Error("WARN_RMQ-PUBLISH-FLOW_START-NOTIFY-FLOW-HANDLER", fmt.Errorf("resuming publishing due to flow request from server"))
		}
		publisher.disablePublishDueToFlowMux.Unlock()
	}
}

func (publisher *Publisher) startNotifyBlockedHandler() {
	blockings := publisher.connManager.NotifyBlockedSafe(make(chan amqp.Blocking))
	publisher.disablePublishDueToBlockedMux.Lock()
	publisher.disablePublishDueToBlocked = false
	publisher.disablePublishDueToBlockedMux.Unlock()

	for b := range blockings {
		publisher.disablePublishDueToBlockedMux.Lock()
		if b.Active {
			publisher.options.Logger.Error("WARN_RMQ-PUBLISH-FLOW_START-NOTIFY-BLOCKED-HANDLER", fmt.Errorf("pausing publishing due to TCP blocking from server"))
			publisher.disablePublishDueToBlocked = true
		} else {
			publisher.disablePublishDueToBlocked = false
			publisher.options.Logger.Error("WARN_RMQ-PUBLISH-FLOW_START-NOTIFY-BLOCKED-HANDLER", fmt.Errorf("resuming publishing due to TCP blocking from server"))
		}
		publisher.disablePublishDueToBlockedMux.Unlock()
	}
}
