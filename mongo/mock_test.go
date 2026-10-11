package mongo

import (
	"context"
	"os"
	"path/filepath"
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
