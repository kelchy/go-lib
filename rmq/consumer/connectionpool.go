package consumer

import (
	"github.com/streadway/amqp"
)

// IConnectionPool interface
type IConnectionPool interface {
	getCon() (*amqp.Connection, error)
}

type connectionPool struct {
	uris            []string
	currentURIIndex int
}

func newConnectionPool(uris ...string) IConnectionPool {
	return &connectionPool{
		currentURIIndex: 0,
		uris:            uris,
	}
}

func (connPool *connectionPool) nextURI() (uri string) {
	if connPool.currentURIIndex == len(connPool.uris)-1 {
		uri = connPool.uris[connPool.currentURIIndex]
		connPool.currentURIIndex = 0
		return
	}
	uri = connPool.uris[connPool.currentURIIndex]
	connPool.currentURIIndex++
	return
}

func (connPool *connectionPool) getCon() (*amqp.Connection, error) {
	var err error
	uri := connPool.nextURI()
	con, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}
	return con, err
}
