package mongo

import (
	"context"
	"net/url"
	"strings"

	"github.com/kelchy/go-lib/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client - instance initiated by constructor
type Client struct {
	Uri        string
	Db         *mongo.Database
	Connection *mongo.Client
	log        log.Log
}

// New - constructor to initiate client instance
func New(uri string) (Client, error) {
	l, _ := log.New("")
	var client Client
	var e error
	// create a client using the mongo uri
	conn, e := mongo.NewClient(options.Client().ApplyURI(uri))
	if e != nil {
		l.Error("MONGO_NEW", e)
		return client, e
	}
	client.Uri = uri
	client.Connection = conn

	// attempt to connect
	e = conn.Connect(context.Background())
	if e != nil {
		l.Error("MONGO_CONNECT", e)
		return client, e
	}

	// after connecting and not seeing any error, attempt to ping
	e = conn.Ping(context.Background(), readpref.Primary())
	if e != nil {
		l.Error("MONGO_PING", e)
		return client, e
	}
	// success, let's assign
	// get db name from uri
	db, e := uri2db(client.Uri)
	if e != nil {
		l.Error("MONGO_URI2DB", e)
		return client, e
	}
	client.Db = conn.Database(db)
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
