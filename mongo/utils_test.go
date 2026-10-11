package mongo

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestTransformHelpers(t *testing.T) {
	if got := IntDateTime(bson.DateTime(123)); got != 123 {
		t.Fatalf("IntDateTime() = %d, want 123", got)
	}
	if got := MapInterface(bson.M{"key": "value"}); got["key"] != "value" {
		t.Fatalf("MapInterface() = %#v", got)
	}
	if got := SliceInterface(bson.A{"value"}); len(got) != 1 || got[0] != "value" {
		t.Fatalf("SliceInterface() = %#v", got)
	}
	if len(Operations) != 4 {
		t.Fatalf("Operations has %d entries, want 4", len(Operations))
	}
}

func TestUnmarshal(t *testing.T) {
	got := Unmarshal(`{"name":"Ada","count":2}`)
	if got["name"] != "Ada" || got["count"] != float64(2) {
		t.Fatalf("Unmarshal() = %#v", got)
	}
	if got := Unmarshal(`{`); len(got) != 0 {
		t.Fatalf("Unmarshal(invalid JSON) = %#v, want empty map", got)
	}
}

func TestSetContext(t *testing.T) {
	nilCtx, nilCancel := SetContext(nil, time.Minute) //nolint:staticcheck // Exercise the helper's nil-context fallback.
	if _, ok := nilCtx.Deadline(); !ok {
		t.Fatal("SetContext(nil) did not add a deadline")
	}
	nilCancel()
	ctx, cancel := SetContext(context.TODO(), time.Minute)
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("SetContext(nil) did not set a deadline")
	}
	cancel()
	parent, parentCancel := context.WithTimeout(context.Background(), time.Hour)
	defer parentCancel()
	ctx, cancel = SetContext(parent, time.Minute)
	defer cancel()
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("SetContext(parent) did not preserve a deadline")
	}
	deadlineParent, deadlineCancel := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
	defer deadlineCancel()
	ctx, cancel = SetContext(deadlineParent, time.Minute)
	defer cancel()
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("SetContext(deadline parent) lost its deadline")
	}
}
