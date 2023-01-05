package mongo

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// Unmarshal - bson unmarshal
func Unmarshal(s string) bson.M {
	var bsonMap bson.M
	e := json.Unmarshal([]byte(s), &bsonMap)
	if e != nil {
		log.Fatal("json. Unmarshal() ERROR:", e)
		return bson.M{}
	}
	return bsonMap
}

// SetContext - helper to set deadline to context
func SetContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, deadlinePresent := ctx.Deadline(); deadlinePresent {
		return context.WithCancel(ctx)
	}
	// context.WithTimeout will return a derived context with timeout <= parent context timeout
	return context.WithTimeout(ctx, timeout)
}
