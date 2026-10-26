package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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
		opts.SetSort(bson.D{{Key: opt["sort"].(string), Value: opt["order"].(int64)}})
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
	filter map[string]interface{}, opt map[string]interface{}) (*SingleResult, error) {
	// select collection
	col := client.Db.Collection(colname)

	// handle options
	opts := options.FindOne()
	if opt["skip"] != nil {
		opts.SetSkip(opt["skip"].(int64))
	}
	if opt["sort"] != nil && opt["order"] != nil {
		opts.SetSort(bson.D{{Key: opt["sort"].(string), Value: opt["order"].(int64)}})
	}

	// fetch doc
	doc := col.FindOne(ctx, filter, opts)
	if doc.Err() != nil {
		// a miss is returned as ErrNoDocuments but not logged
		if !expected(doc.Err()) {
			client.log.Error("MONGO_FINDONE", doc.Err())
		}
		return nil, doc.Err()
	}

	return NewSingleResult(doc), nil
}

// CountDocuments counts the docs in collection that match filter. Pass
// options.Count().SetLimit(1) to only check whether any doc matches.
func (client Client) CountDocuments(ctx context.Context, colname string, filter interface{},
	opts ...options.Lister[options.CountOptions]) (int64, error) {
	n, err := client.Db.Collection(colname).CountDocuments(ctx, filter, opts...)
	if err != nil {
		client.log.Error("MONGO_COUNTDOCUMENTS", err)
	}
	return n, err
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
