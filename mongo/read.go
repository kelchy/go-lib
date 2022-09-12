package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Find - function to find doc in collection
//	transactionCtx is only required for transactions, you can pass nil for normal usage
func (client Client) Find(transactionCtx context.Context, colname string, filter map[string]interface{},
	opt map[string]interface{}) ([]bson.M, error) {
	// select collection
	col := client.db.Collection(colname)

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

	// set context
	ctx, cancel := SetContext(transactionCtx, client.timeout)
	defer cancel()

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

// FindOne - function to find first encountered doc in collection
//	transactionCtx is only required for transactions, you can pass nil for normal usage
func (client Client) FindOne(transactionCtx context.Context, colname string,
	filter map[string]interface{}, opt map[string]interface{}) (*mongo.SingleResult, error) {
	// select collection
	col := client.db.Collection(colname)

	// handle options
	opts := options.FindOne()
	if opt["skip"] != nil {
		opts.SetSkip(opt["skip"].(int64))
	}
	if opt["sort"] != nil && opt["order"] != nil {
		opts.SetSort(bson.D{{opt["sort"].(string), opt["order"].(int64)}})
	}

	// set context
	ctx, cancel := SetContext(transactionCtx, client.timeout)
	defer cancel()

	// fetch doc
	doc := col.FindOne(ctx, filter, opts)
	if doc.Err() != nil {
		client.log.Error("MONGO_FINDONE", doc.Err())
		return nil, doc.Err()
	}

	return doc, nil
}

// Aggregate - function to aggregate docs in collection
//		transactionCtx is only required for transactions, you can pass nil for normal usage
func (client Client) Aggregate(transactionCtx context.Context, colname string, pipeline []interface{}) ([]bson.M, error) {
	// select collection
	col := client.db.Collection(colname)

	var e error
	var docs []bson.M

	// set context
	ctx, cancel := SetContext(transactionCtx, client.timeout)
	defer cancel()

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
