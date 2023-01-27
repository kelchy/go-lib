package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Find - function to find doc in collection, ctx can be nil
func (client Client) Find(ctx context.Context, colname string, filter map[string]interface{},
	opt map[string]interface{}) ([]bson.M, error) {
	// select collection
	col := client.Db.Collection(colname)

	// handle options
	opts := options.Find()
	if opt["skip"] != nil {
		opts.SetSkip(opt["skip"].(int64))
	}
	if opt["limit"] != nil {
		opts.SetLimit(opt["limit"].(int64))
	}
	if opt["sort"] != nil && opt["order"] != nil {
		opts.SetSort(bson.D{{opt["sort"].(string), opt["order"].(int64)}})
	}

	var e error
	var docs []bson.M

	// fetch cursor
	cursor, e := col.Find(ctx, filter, opts)
	if e != nil {
		client.log.Error("MONGO_FIND", e)
		return docs, e
	}

	// move cursor to fetch all
	e = cursor.All(ctx, &docs)
	if e != nil {
		client.log.Error("MONGO_CURSOR", e)
		return docs, e
	}

	return docs, e
}

// FindOne - function to find first encountered doc in collection, ctx can be nil
func (client Client) FindOne(ctx context.Context, colname string,
	filter map[string]interface{}, opt map[string]interface{}) (*mongo.SingleResult, error) {
	// select collection
	col := client.Db.Collection(colname)

	// handle options
	opts := options.FindOne()
	if opt["skip"] != nil {
		opts.SetSkip(opt["skip"].(int64))
	}
	if opt["sort"] != nil && opt["order"] != nil {
		opts.SetSort(bson.D{{opt["sort"].(string), opt["order"].(int64)}})
	}

	// fetch doc
	doc := col.FindOne(ctx, filter, opts)
	if doc.Err() != nil {
		client.log.Error("MONGO_FINDONE", doc.Err())
		return nil, doc.Err()
	}

	return doc, nil
}

// Aggregate - function to aggregate docs in collection, ctx can be nil
func (client Client) Aggregate(ctx context.Context, colname string, pipeline []interface{}) ([]bson.M, error) {
	// select collection
	col := client.Db.Collection(colname)

	var e error
	var docs []bson.M

	// fetch cursor
	cursor, e := col.Aggregate(ctx, pipeline)
	if e != nil {
		client.log.Error("MONGO_AGGREGATE", e)
		return docs, e
	}

	// move cursor to fetch all
	e = cursor.All(ctx, &docs)
	if e != nil {
		client.log.Error("MONGO_AGGREGATE_CURSOR", e)
		return docs, e
	}

	return docs, e
}
