package mongo

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/kelchy/go-lib/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client - instance initiated by constructor
type Client struct {
	uri		string
	db		*mongo.Database
	connection	*mongo.Client
	timeout		time.Duration
	log		log.Log
}

// New - constructor to initiate client instance
func New(uri string, timeout int) (Client, error) {	
        l, _ := log.New("")
	var client Client
	var e error
	// create a client using the mongo uri
	conn, e := mongo.NewClient(options.Client().ApplyURI(uri))
	if e != nil {
		l.Error("MONGO_NEW", e)
		return client, e
	}
	client.uri = uri
	client.connection = conn
	client.timeout = time.Duration(timeout) * time.Second
	// set a context timer "ctxTimeout" to make sure we don't
	// wait indefinitely for the connection to happen
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	// attempt to connect
	e = conn.Connect(ctx)
	if e != nil {
		l.Error("MONGO_CONNECT", e)
		return client, e
	}

	// after connecting and not seeing any error, attempt to ping
	ctx, cancel = context.WithTimeout(context.Background(), client.timeout)
	defer cancel()
	e = conn.Ping(ctx, readpref.Primary())
	if e != nil {
		l.Error("MONGO_PING", e)
		return client, e
	}
	// success, let's assign
	// get db name from uri
	db, e := uri2db(client.uri)
	if e != nil {
		l.Error("MONGO_URI2DB", e)
		return client, e
	}
	client.db = conn.Database(db)
	client.log = l
	return client, e
}

// function to parse the db name string from the var uri
// assuming uri is something valid as Ping() was done before calling this
func uri2db(uri string) (string, error) {
	u, e := url.Parse(uri)
	// path may have a leading /
	return strings.TrimLeft(u.Path, "/"), e
}
