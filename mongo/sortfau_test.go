package mongo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
)

// watched connects a client to a mock deployment and records the commands it
// sends, by name.
func watched(t *testing.T, responses ...bson.D) (Client, map[string][]bson.Raw) {
	t.Helper()
	sent := map[string][]bson.Raw{}
	clientOptions := options.Client().SetMonitor(&event.CommandMonitor{Started: func(_ context.Context, e *event.CommandStartedEvent) {
		sent[e.CommandName] = append(sent[e.CommandName], e.Command)
	}})
	clientOptions.Deployment = drivertest.NewMockDeployment(responses...) //nolint:staticcheck // Driver test hook required for an in-memory deployment.
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		t.Fatalf("connect mock MongoDB client: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	return Client{Db: client.Database("test")}, sent
}

// sortKeys returns the keys of a sent command's sort document, in order.
func sortKeys(t *testing.T, cmd bson.Raw) []string {
	t.Helper()
	elems, err := cmd.Lookup("sort").Document().Elements()
	if err != nil {
		t.Fatalf("sort: %v", err)
	}
	keys := []string{}
	for _, e := range elems {
		keys = append(keys, fmt.Sprintf("%s=%d", e.Key(), e.Value().AsInt64()))
	}
	return keys
}

func TestSortOption(t *testing.T) {
	two := bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}
	tests := []struct {
		name string
		opt  map[string]interface{}
		want bson.D
		ok   bool
	}{
		{"none", nil, nil, false},
		{"single", map[string]interface{}{"sort": "name", "order": int64(1)}, bson.D{{Key: "name", Value: int64(1)}}, true},
		{"sort without order", map[string]interface{}{"sort": "name"}, nil, false},
		{"several", map[string]interface{}{"sorts": two}, two, true},
		{"several wins", map[string]interface{}{"sorts": two, "sort": "name", "order": int64(1)}, two, true},
		{"empty sorts falls back", map[string]interface{}{"sorts": bson.D{}, "sort": "name", "order": int64(1)}, bson.D{{Key: "name", Value: int64(1)}}, true},
	}
	for _, tt := range tests {
		got, ok := sortOption(tt.opt)
		if ok != tt.ok || len(got) != len(tt.want) {
			t.Fatalf("%s: sortOption() = (%v, %v), want (%v, %v)", tt.name, got, ok, tt.want, tt.ok)
		}
		for i := range got {
			if got[i].Key != tt.want[i].Key || got[i].Value != tt.want[i].Value {
				t.Fatalf("%s: sortOption()[%d] = %v, want %v", tt.name, i, got[i], tt.want[i])
			}
		}
	}
}

func TestFindSortsSendsKeysInOrder(t *testing.T) {
	c, sent := watched(t, emptyCursor, emptyCursor)
	ctx := context.Background()
	sorts := bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}
	if _, err := c.Find(ctx, "lines", nil, map[string]interface{}{"sorts": sorts, "limit": int64(10)}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.FindOne(ctx, "lines", nil, map[string]interface{}{"sorts": sorts}); err != nil && !errors.Is(err, ErrNoDocuments) {
		t.Fatal(err)
	}
	if len(sent["find"]) != 2 {
		t.Fatalf("sent %d finds, want 2", len(sent["find"]))
	}
	for _, cmd := range sent["find"] {
		if got := strings.Join(sortKeys(t, cmd), ","); got != "created_at=1,_id=1" {
			t.Fatalf("find sort = %s, want created_at=1,_id=1", got)
		}
	}
}

func TestFindOneAndUpdate(t *testing.T) {
	claimed := bson.D{{Key: "ok", Value: 1}, {Key: "value", Value: bson.D{{Key: "_id", Value: "d1"}, {Key: "state", Value: "sending"}}},
		{Key: "lastErrorObject", Value: bson.D{{Key: "n", Value: 1}, {Key: "updatedExisting", Value: true}}}}
	miss := bson.D{{Key: "ok", Value: 1}, {Key: "value", Value: nil},
		{Key: "lastErrorObject", Value: bson.D{{Key: "n", Value: 0}, {Key: "updatedExisting", Value: false}}}}
	c, sent := watched(t, claimed, miss, duplicateKey, commandError)
	ctx := context.Background()
	update := bson.M{"$set": bson.M{"state": "sending"}}
	opts := options.FindOneAndUpdate().SetSort(bson.D{{Key: "due", Value: 1}, {Key: "_id", Value: 1}}).
		SetReturnDocument(options.After).SetUpsert(true)

	// a match comes back, updated
	res, err := c.FindOneAndUpdate(ctx, "deliveries", bson.M{"state": "queued"}, update, opts)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		ID    string `bson:"_id"`
		State string `bson:"state"`
	}
	if err := res.Decode(&doc); err != nil || doc.ID != "d1" || doc.State != "sending" {
		t.Fatalf("FindOneAndUpdate() = (%+v, %v), want d1 sending", doc, err)
	}
	cmd := sent["findAndModify"][0]
	if got := strings.Join(sortKeys(t, cmd), ","); got != "due=1,_id=1" {
		t.Fatalf("sort = %s, want due=1,_id=1", got)
	}
	if !cmd.Lookup("new").Boolean() || !cmd.Lookup("upsert").Boolean() {
		t.Fatalf("sent %v, want new and upsert true", cmd)
	}

	// no match: ErrNoDocuments, not logged
	logged := stderr(t, func() { _, err = c.FindOneAndUpdate(ctx, "deliveries", bson.M{"state": "queued"}, update) })
	if !errors.Is(err, ErrNoDocuments) || strings.Contains(logged, "MONGO_") {
		t.Fatalf("FindOneAndUpdate(miss) = %v, logged %q; want an unlogged ErrNoDocuments", err, logged)
	}
	// a duplicate key on upsert is expected too
	logged = stderr(t, func() { _, err = c.FindOneAndUpdate(ctx, "deliveries", bson.M{"_id": "d1"}, update, opts) })
	if !IsDuplicateKeyError(err) || strings.Contains(logged, "MONGO_") {
		t.Fatalf("FindOneAndUpdate(dup) = %v, logged %q; want an unlogged duplicate key", err, logged)
	}
	// a real failure is returned and logged
	logged = stderr(t, func() { _, err = c.FindOneAndUpdate(ctx, "deliveries", bson.M{}, update) })
	if err == nil || !strings.Contains(logged, "MONGO_FINDONEANDUPDATE") {
		t.Fatalf("FindOneAndUpdate(fail) = %v, logged %q; want a logged error", err, logged)
	}
}
