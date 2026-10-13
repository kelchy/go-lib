package mongo

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

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
