package rabbitmq

import (
	"github.com/kelchy/go-lib/rmq/consumer/internal/channelmanager"
	"github.com/kelchy/go-lib/rmq/consumer/internal/logger"
)

func declareExchange(chanManager *channelmanager.ChannelManager, options ExchangeOptions) error {
	if !options.Declare {
		return nil
	}
	if options.Passive {
		err := chanManager.ExchangeDeclarePassiveSafe(
			options.Name,
			options.Kind,
			options.Durable,
			options.AutoDelete,
			options.Internal,
			options.NoWait,
			tableToAMQPTable(options.Args),
		)
		if err != nil {
			return err
		}
		return nil
	}
	err := chanManager.ExchangeDeclareSafe(
		options.Name,
		options.Kind,
		options.Durable,
		options.AutoDelete,
		options.Internal,
		options.NoWait,
		tableToAMQPTable(options.Args),
	)
	if err != nil {
		return err
	}
	return nil
}

func declareQueue(chanManager *channelmanager.ChannelManager, options QueueOptions) error {
	if !options.Declare {
		return nil
	}
	if options.Passive {
		_, err := chanManager.QueueDeclarePassiveSafe(
			options.Name,
			options.Durable,
			options.AutoDelete,
			options.Exclusive,
			options.NoWait,
			tableToAMQPTable(options.Args),
		)
		if err != nil {
			return err
		}
		return nil
	}
	_, err := chanManager.QueueDeclareSafe(
		options.Name,
		options.Durable,
		options.AutoDelete,
		options.Exclusive,
		options.NoWait,
		tableToAMQPTable(options.Args),
	)
	if err != nil {
		return err
	}
	return nil
}

func declareBindings(chanManager *channelmanager.ChannelManager, options ConsumerOptions) error {
	for _, binding := range options.Bindings {
		if !binding.Declare {
			continue
		}
		err := chanManager.QueueBindSafe(
			options.QueueOptions.Name,
			binding.RoutingKey,
			options.ExchangeOptions.Name,
			binding.NoWait,
			tableToAMQPTable(binding.Args),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeclareExchange declares all exchanges in the given options
func DeclareExchange(
	conn *Conn,
	options ExchangeOptions,
) error {
	// Creates a new channel manager
	chanManager, err := channelmanager.NewChannelManager(conn.connectionManager, logger.DefaultLogger, conn.connectionManager.ReconnectInterval)
	if err != nil {
		return err
	}
	// Close the channel manager when done
	defer chanManager.Close()
	return declareExchange(chanManager, options)
}

// DeclareQueue declares the queue in the given options
func DeclareQueue(
	conn *Conn,
	options QueueOptions,
) error {
	// Creates a new channel manager
	chanManager, err := channelmanager.NewChannelManager(conn.connectionManager, logger.DefaultLogger, conn.connectionManager.ReconnectInterval)
	if err != nil {
		return err
	}
	// Close the channel manager when done
	defer chanManager.Close()
	return declareQueue(chanManager, options)
}

// DeclareBinding declares the binding in the given options
func DeclareBinding(
	conn *Conn,
	options BindingDeclareOptions,
) error {
	// Creates a new channel manager
	chanManager, err := channelmanager.NewChannelManager(conn.connectionManager, logger.DefaultLogger, conn.connectionManager.ReconnectInterval)
	if err != nil {
		return err
	}
	// Close the channel manager when done
	defer chanManager.Close()
	if err := chanManager.QueueBindSafe(
		options.QueueName,
		options.RoutingKey,
		options.ExchangeName,
		options.NoWait,
		tableToAMQPTable(options.Args),
	); err != nil {
		return err
	}
	return nil
}
