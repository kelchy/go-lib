package mongo

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
)

// mockMongo creates a client backed by the driver's in-memory wire-protocol
// deployment. Queue command responses on deployment before invoking operations.
func mockMongo(t *testing.T, responses ...bson.D) (*mongo.Client, *drivertest.MockDeployment) {
	t.Helper()
	deployment := drivertest.NewMockDeployment(responses...)
	clientOptions := options.Client()
	clientOptions.Deployment = deployment //nolint:staticcheck // Driver test hook required for an in-memory deployment.
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		t.Fatalf("connect mock MongoDB client: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	return client, deployment
}

func TestMockMongoFind(t *testing.T) {
	client, _ := mockMongo(t, bson.D{{Key: "ok", Value: 1}, {Key: "cursor", Value: bson.D{
		{Key: "id", Value: int64(0)},
		{Key: "ns", Value: "test.items"},
		{Key: "firstBatch", Value: bson.A{bson.D{{Key: "name", Value: "Ada"}}}},
	}}})
	docs, err := (Client{Db: client.Database("test")}).Find(context.Background(), "items", nil, nil)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if len(docs) != 1 || docs[0]["name"] != "Ada" {
		t.Fatalf("Find() = %#v, want one Ada document", docs)
	}
}

func TestMockMongoCRUD(t *testing.T) {
	responses := []bson.D{
		{{Key: "ok", Value: 1}, {Key: "cursor", Value: bson.D{{Key: "id", Value: int64(0)}, {Key: "ns", Value: "test.items"}, {Key: "firstBatch", Value: bson.A{}}}}},
		{{Key: "ok", Value: 1}, {Key: "cursor", Value: bson.D{{Key: "id", Value: int64(0)}, {Key: "ns", Value: "test.items"}, {Key: "firstBatch", Value: bson.A{bson.D{{Key: "name", Value: "Ada"}}}}}}},
		{{Key: "ok", Value: 1}, {Key: "cursor", Value: bson.D{{Key: "id", Value: int64(0)}, {Key: "ns", Value: "test.items"}, {Key: "firstBatch", Value: bson.A{}}}}},
		{{Key: "ok", Value: 1}, {Key: "n", Value: 2}, {Key: "nModified", Value: 2}},
		{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "nModified", Value: 1}},
		{{Key: "ok", Value: 1}, {Key: "n", Value: 2}},
		{{Key: "ok", Value: 1}, {Key: "n", Value: 1}},
		{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "nModified", Value: 1}},
		{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "nModified", Value: 1}},
		{{Key: "ok", Value: 1}, {Key: "n", Value: 1}},
		{{Key: "ok", Value: 1}, {Key: "n", Value: 1}},
		{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "nModified", Value: 1}},
	}
	client, _ := mockMongo(t, responses...)
	c := Client{Db: client.Database("test")}
	ctx := context.Background()
	if _, err := c.Find(ctx, "items", nil, map[string]interface{}{"skip": int64(1), "limit": int64(2), "sort": "name", "order": int64(1)}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.FindOne(ctx, "items", nil, map[string]interface{}{"skip": int64(1), "sort": "name", "order": int64(1)}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Aggregate(ctx, "items", []interface{}{bson.D{{Key: "$match", Value: bson.D{}}}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.InsertMany(ctx, "items", []interface{}{bson.D{{Key: "a", Value: 1}}}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.InsertOne(ctx, "items", bson.D{{Key: "a", Value: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateMany(ctx, "items", bson.D{}, bson.D{{Key: "$set", Value: bson.D{{Key: "a", Value: 2}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateOne(ctx, "items", bson.D{}, bson.D{{Key: "$set", Value: bson.D{{Key: "a", Value: 2}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DeleteMany(ctx, "items", bson.D{}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DeleteOne(ctx, "items", bson.D{}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReplaceOne(ctx, "items", bson.D{}, bson.D{{Key: "a", Value: 3}}); err != nil {
		t.Fatal(err)
	}
}

func TestMockMongoObjectIDReturns(t *testing.T) {
	id := bson.NewObjectID()
	client, _ := mockMongo(t,
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "insertedIds", Value: bson.A{id}}},
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "insertedId", Value: id}},
	)
	c := Client{Db: client.Database("test")}
	count, ids, err := c.InsertMany(context.Background(), "items", []interface{}{bson.D{{Key: "_id", Value: id}, {Key: "x", Value: 1}}}, true)
	if err != nil || count != 1 || ids[0] != id.Hex() {
		t.Fatalf("InsertMany() = (%d, %#v, %v), want (%d, [%q], nil)", count, ids, err, 1, id.Hex())
	}
	gotID, err := c.InsertOne(context.Background(), "items", bson.D{{Key: "_id", Value: id}, {Key: "x", Value: 1}})
	if err != nil || gotID != id.Hex() {
		t.Fatalf("InsertOne() = (%v, %v), want (%q, nil)", gotID, err, id.Hex())
	}
}

func TestMockMongoCommandErrors(t *testing.T) {
	ctx := context.Background()
	errResponse := bson.D{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "mock command failure"}, {Key: "code", Value: int32(123)}}
	tests := []struct {
		name string
		call func(Client) error
	}{
		{"find", func(c Client) error { _, err := c.Find(ctx, "items", nil, nil); return err }},
		{"find one", func(c Client) error { _, err := c.FindOne(ctx, "items", nil, nil); return err }},
		{"aggregate", func(c Client) error { _, err := c.Aggregate(ctx, "items", nil); return err }},
		{"insert many", func(c Client) error {
			_, _, err := c.InsertMany(ctx, "items", []interface{}{bson.D{{Key: "x", Value: 1}}}, true)
			return err
		}},
		{"insert one", func(c Client) error { _, err := c.InsertOne(ctx, "items", bson.D{{Key: "x", Value: 1}}); return err }},
		{"update many", func(c Client) error {
			_, err := c.UpdateMany(ctx, "items", bson.D{}, bson.D{{Key: "$set", Value: bson.D{{Key: "x", Value: 2}}}})
			return err
		}},
		{"update one", func(c Client) error {
			_, err := c.UpdateOne(ctx, "items", bson.D{}, bson.D{{Key: "$set", Value: bson.D{{Key: "x", Value: 2}}}})
			return err
		}},
		{"delete many", func(c Client) error { _, err := c.DeleteMany(ctx, "items", bson.D{}); return err }},
		{"delete one", func(c Client) error { _, err := c.DeleteOne(ctx, "items", bson.D{}); return err }},
		{"replace one", func(c Client) error {
			_, err := c.ReplaceOne(ctx, "items", bson.D{}, bson.D{{Key: "x", Value: 2}})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := mockMongo(t, errResponse)
			if err := tt.call(Client{Db: client.Database("test")}); err == nil {
				t.Fatal("expected command error")
			}
		})
	}
}

func TestMockMongoClientHelpers(t *testing.T) {
	client, _ := mockMongo(t,
		bson.D{{Key: "ok", Value: 1}}, // Ping
		bson.D{{Key: "ok", Value: 1}}, // Create index
	)
	c := Client{Db: client.Database("test"), Connection: client}
	if err := c.Ready(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.CreateTTLIndex(context.Background(), "items", "created", 60); err != nil {
		t.Fatal(err)
	}
}

func TestMongoLocalErrorPaths(t *testing.T) {
	if _, err := New("not a mongodb uri"); err == nil {
		t.Fatal("New() expected invalid URI error")
	}
	if _, err := NewContext(context.Background(), "mongodb://localhost:27017"); err == nil {
		t.Fatal("NewContext() expected missing database error")
	}

	missingFile := filepath.Join(t.TempDir(), "missing")
	mockClient, _ := mockMongo(t)
	if _, err := (Client{Db: mockClient.Database("test")}).FSset(missingFile); err == nil {
		t.Fatal("FSset() expected file read error")
	}

	parent, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()
	ctx, cancel := SetContext(parent, time.Minute)
	defer cancel()
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("SetContext() did not add a deadline")
	}

	file := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(file, []byte("contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	client, _ := mockMongo(t, bson.D{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "mock upload failure"}, {Key: "code", Value: int32(123)}})
	if _, err := (Client{Db: client.Database("test")}).FSset(file); err == nil {
		t.Fatal("FSset() expected mock upload error")
	}
}

func TestNewClientWithMockDeployment(t *testing.T) {
	deployment := drivertest.NewMockDeployment(bson.D{{Key: "ok", Value: 1}})
	clientOptions := options.Client()
	clientOptions.Deployment = deployment //nolint:staticcheck // Driver test hook required for an in-memory deployment.
	client, err := newClientWithOptions(context.Background(), "mongodb://localhost:27017/test", true, clientOptions)
	if err != nil {
		t.Fatalf("newClientWithOptions() error = %v", err)
	}
	if client.URI != "mongodb://localhost:27017/test" || client.Db == nil || client.Connection == nil {
		t.Fatalf("newClientWithOptions() returned incomplete client: %#v", client)
	}
	if err := client.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	failedDeployment := drivertest.NewMockDeployment(bson.D{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "ping failed"}, {Key: "code", Value: int32(123)}})
	failedOptions := options.Client()
	failedOptions.Deployment = failedDeployment //nolint:staticcheck // Driver test hook required to simulate a failed ping.
	if _, err := newClientWithOptions(context.Background(), "mongodb://localhost:27017/test", true, failedOptions); err == nil {
		t.Fatal("newClientWithOptions() expected ping error")
	}
}

func TestMockMongoGridFSAndClose(t *testing.T) {
	ctx := context.Background()
	listClient, _ := mockMongo(t, bson.D{{Key: "ok", Value: 1}, {Key: "cursor", Value: bson.D{
		{Key: "id", Value: int64(0)}, {Key: "ns", Value: "test.fs.files"}, {Key: "firstBatch", Value: bson.A{}},
	}}})
	c := Client{Db: listClient.Database("test")}
	if _, err := c.FSlist(ctx, bson.M{}, "filename", 1, 0, 10); err != nil {
		t.Fatalf("FSlist() error = %v", err)
	}

	getClient, _ := mockMongo(t, bson.D{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "missing file"}, {Key: "code", Value: int32(26)}})
	if _, err := (Client{Db: getClient.Database("test")}).FSget("missing"); err == nil {
		t.Fatal("FSget() expected download error")
	}

	closeDeployment := drivertest.NewMockDeployment()
	options := options.Client()
	options.Deployment = closeDeployment //nolint:staticcheck // Driver test hook required for an in-memory deployment.
	closeClient, err := mongo.Connect(options)
	if err != nil {
		t.Fatal(err)
	}
	if err := (Client{Connection: closeClient}).Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestMockMongoTransaction(t *testing.T) {
	client, _ := mockMongo(t,
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "nModified", Value: 1}},
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "nModified", Value: 1}},
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "nModified", Value: 1}},
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 2}, {Key: "insertedIds", Value: bson.A{"one", "two"}}},
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "insertedId", Value: "custom-id"}},
		bson.D{{Key: "ok", Value: 1}},
	)
	c := Client{Db: client.Database("test"), Connection: client}
	result, err := c.Transaction([]map[string]interface{}{
		{"operation": "updateOne", "collection": "items", "filter": bson.D{}, "update": bson.D{{Key: "$set", Value: bson.D{{Key: "x", Value: 2}}}}},
		{"operation": "updateMany", "collection": "items", "filter": bson.D{}, "update": bson.D{{Key: "$set", Value: bson.D{{Key: "x", Value: 2}}}}},
		{"operation": "insertOne", "collection": "items", "doc": bson.D{{Key: "x", Value: 1}}},
		{"operation": "insertMany", "collection": "items", "ordered": true, "docs": []interface{}{bson.D{{Key: "x", Value: 1}}, bson.D{{Key: "x", Value: 2}}}},
	}, 10)
	if err != nil {
		t.Fatalf("Transaction() error = %v", err)
	}
	if len(result.([]interface{})) != 4 {
		t.Fatalf("Transaction() result = %#v", result)
	}
	if _, err := c.Transaction([]map[string]interface{}{{"operation": "unsupported"}}, 10); err == nil {
		t.Fatal("Transaction() expected unknown operation error")
	}
}

// duplicateKey is the server's reply when a write violates a unique index.
var duplicateKey = bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 0}, {Key: "writeErrors", Value: bson.A{
	bson.D{{Key: "index", Value: 0}, {Key: "code", Value: int32(11000)}, {Key: "errmsg", Value: "E11000 duplicate key error"}},
}}}

// commandError is a failure that is not an expected outcome and must be logged.
var commandError = bson.D{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "mock command failure"}, {Key: "code", Value: int32(123)}}

// emptyCursor is a find reply with no documents.
var emptyCursor = bson.D{{Key: "ok", Value: 1}, {Key: "cursor", Value: bson.D{
	{Key: "id", Value: int64(0)}, {Key: "ns", Value: "test.items"}, {Key: "firstBatch", Value: bson.A{}},
}}}

// stderr runs fn and returns what it wrote to stderr, which is where the
// go-lib logger prints errors.
func stderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()
	fn()
	_ = w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestIsDuplicateKeyError(t *testing.T) {
	client, _ := mockMongo(t, duplicateKey)
	_, err := client.Database("test").Collection("items").InsertOne(context.Background(), bson.D{{Key: "_id", Value: 1}})
	if !IsDuplicateKeyError(err) {
		t.Fatalf("IsDuplicateKeyError(%v) = false, want true", err)
	}
	if IsDuplicateKeyError(errors.New("other")) || IsDuplicateKeyError(nil) {
		t.Fatal("IsDuplicateKeyError() = true for a non-duplicate error")
	}
}

func TestCountDocuments(t *testing.T) {
	// CountDocuments runs an aggregate whose single result holds the count
	client, _ := mockMongo(t,
		bson.D{{Key: "ok", Value: 1}, {Key: "cursor", Value: bson.D{
			{Key: "id", Value: int64(0)}, {Key: "ns", Value: "test.items"},
			{Key: "firstBatch", Value: bson.A{bson.D{{Key: "n", Value: int32(1)}}}},
		}}},
		commandError,
	)
	c := Client{Db: client.Database("test")}
	n, err := c.CountDocuments(context.Background(), "items", bson.M{"a": 1}, options.Count().SetLimit(1))
	if err != nil || n != 1 {
		t.Fatalf("CountDocuments() = (%d, %v), want (1, nil)", n, err)
	}
	if _, err := c.CountDocuments(context.Background(), "items", bson.M{}); err == nil {
		t.Fatal("CountDocuments() expected command error")
	}
}

func TestCreateIndex(t *testing.T) {
	client, _ := mockMongo(t, bson.D{{Key: "ok", Value: 1}}, commandError)
	c := Client{Db: client.Database("test")}
	keys := bson.D{{Key: "call_id", Value: 1}, {Key: "seq", Value: 1}}
	// the driver derives the name from the keys in order
	name, err := c.CreateIndex(context.Background(), "items", keys, options.Index().SetUnique(true))
	if err != nil || name != "call_id_1_seq_1" {
		t.Fatalf("CreateIndex() = (%q, %v), want (call_id_1_seq_1, nil)", name, err)
	}
	if _, err := c.CreateIndex(context.Background(), "items", keys, nil); err == nil {
		t.Fatal("CreateIndex() expected command error")
	}
}

func TestUpdateResultReturnsMatchedCount(t *testing.T) {
	// n is matched, nModified is modified: one doc matched, nothing changed
	client, _ := mockMongo(t,
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 1}, {Key: "nModified", Value: 0}},
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 3}, {Key: "nModified", Value: 2}},
	)
	c := Client{Db: client.Database("test")}
	ctx := context.Background()
	unset := bson.M{"$unset": bson.M{"absent": ""}}

	one, err := c.UpdateOneResult(ctx, "items", bson.M{"_id": 1}, unset, options.UpdateOne().SetUpsert(false))
	if err != nil || one.MatchedCount != 1 || one.ModifiedCount != 0 {
		t.Fatalf("UpdateOneResult() = (%+v, %v), want matched 1, modified 0", one, err)
	}
	many, err := c.UpdateManyResult(ctx, "items", bson.M{}, unset)
	if err != nil || many.MatchedCount != 3 || many.ModifiedCount != 2 {
		t.Fatalf("UpdateManyResult() = (%+v, %v), want matched 3, modified 2", many, err)
	}
}

// Expected outcomes are returned to the caller but not logged; real failures
// are both returned and logged.
func TestExpectedErrorsAreNotLogged(t *testing.T) {
	ctx := context.Background()
	set := bson.M{"$set": bson.M{"a": 1}}
	tests := []struct {
		name     string
		reply    bson.D
		wantLog  bool
		wantDup  bool
		wantMiss bool
		call     func(Client) error
	}{
		{"find one miss", emptyCursor, false, false, true, func(c Client) error {
			_, err := c.FindOne(ctx, "items", nil, nil)
			return err
		}},
		{"insert one duplicate", duplicateKey, false, true, false, func(c Client) error {
			_, err := c.InsertOne(ctx, "items", bson.D{{Key: "_id", Value: 1}})
			return err
		}},
		{"update one upsert duplicate", duplicateKey, false, true, false, func(c Client) error {
			_, err := c.UpdateOne(ctx, "items", bson.M{"_id": 1}, set, options.UpdateOne().SetUpsert(true))
			return err
		}},
		{"update one result duplicate", duplicateKey, false, true, false, func(c Client) error {
			_, err := c.UpdateOneResult(ctx, "items", bson.M{"_id": 1}, set, options.UpdateOne().SetUpsert(true))
			return err
		}},
		{"update many result duplicate", duplicateKey, false, true, false, func(c Client) error {
			_, err := c.UpdateManyResult(ctx, "items", bson.M{}, set, options.UpdateMany().SetUpsert(true))
			return err
		}},
		{"update one result failure", commandError, true, false, false, func(c Client) error {
			_, err := c.UpdateOneResult(ctx, "items", bson.M{}, set)
			return err
		}},
		{"update many result failure", commandError, true, false, false, func(c Client) error {
			_, err := c.UpdateManyResult(ctx, "items", bson.M{}, set)
			return err
		}},
		{"insert one failure", commandError, true, false, false, func(c Client) error {
			_, err := c.InsertOne(ctx, "items", bson.D{{Key: "_id", Value: 1}})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := mockMongo(t, tt.reply)
			c := Client{Db: client.Database("test")}
			var err error
			logged := stderr(t, func() { err = tt.call(c) })
			if err == nil {
				t.Fatal("expected an error")
			}
			if IsDuplicateKeyError(err) != tt.wantDup || errors.Is(err, ErrNoDocuments) != tt.wantMiss {
				t.Fatalf("error = %v, want duplicate %v, miss %v", err, tt.wantDup, tt.wantMiss)
			}
			if got := strings.Contains(logged, "MONGO_"); got != tt.wantLog {
				t.Fatalf("logged = %v, want %v; stderr: %s", got, tt.wantLog, logged)
			}
		})
	}
}
