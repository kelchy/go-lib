package mongo

import (
	"context"
	"errors"

	"github.com/kelchy/go-lib/log"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

// Client - instance initiated by constructor
type Client struct {
	URI        string
	Db         *mongo.Database
	Connection *mongo.Client
	log        log.Log
}

// ErrNoDocuments is returned by FindOne when a query matches no documents.
var ErrNoDocuments = mongo.ErrNoDocuments

// New - constructor to initiate client instance
func New(uri string) (Client, error) {
	return newClient(context.Background(), uri, false)
}

// NewContext constructs a client, verifies the URI and database, and pings MongoDB.
func NewContext(ctx context.Context, uri string) (Client, error) {
	return newClient(ctx, uri, true)
}

func newClient(ctx context.Context, uri string, requireDatabase bool) (Client, error) {
	return newClientWithOptions(ctx, uri, requireDatabase, options.Client())
}

func newClientWithOptions(ctx context.Context, uri string, requireDatabase bool, clientOptions *options.ClientOptions) (Client, error) {
	l, _ := log.New("")
	var client Client

	cs, err := connstring.ParseAndValidate(uri)
	if err != nil {
		return client, err
	}
	if requireDatabase && cs.Database == "" {
		return client, errors.New("MONGO_URI must name a database")
	}

	conn, err := mongo.Connect(clientOptions.ApplyURI(uri))
	if err != nil {
		l.Error("MONGO_NEW", err)
		return client, err
	}
	client.URI = uri
	client.Connection = conn

	if err = conn.Ping(ctx, readpref.Primary()); err != nil {
		l.Error("MONGO_PING", err)
		_ = conn.Disconnect(context.Background())
		return client, err
	}

	client.Db = conn.Database(cs.Database)
	client.log = l
	return client, nil
}

// Close disconnects the client.
func (client Client) Close(ctx context.Context) error {
	return client.Connection.Disconnect(ctx)
}

// Ready checks that MongoDB is reachable.
func (client Client) Ready(ctx context.Context) error {
	return client.Connection.Ping(ctx, readpref.Primary())
}

// CreateTTLIndex creates a single-field TTL index for a collection.
func (client Client) CreateTTLIndex(ctx context.Context, collection, field string, expireAfterSeconds int32) error {
	_, err := client.Db.Collection(collection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    map[string]int32{field: 1},
		Options: options.Index().SetExpireAfterSeconds(expireAfterSeconds),
	})
	if err != nil {
		client.log.Error("MONGO_CREATE_TTL_INDEX", err)
	}
	return err
}
