package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InsertMany - function to insert many docs into collection, ctx can be nil
func (client Client) InsertMany(ctx context.Context, colname string, docs []interface{}, ordered bool) (
	int, []interface{}, error) {
	// select collection
	col := client.Db.Collection(colname)

	// handle options
	opts := options.InsertManyOptions{}
	if ordered {
		opts.SetOrdered(true)
	} else {
		opts.SetOrdered(false)
	}

	// insert
	/*
	   result is the list of _id that are successfully inserted. you can override the default _id generated by mongo by
	   passing it as one of the key in the document to be inserted
	*/
	result, e := col.InsertMany(ctx, docs, &opts)
	res := []interface{}{}
	if result != nil {
		for _, id := range result.InsertedIDs {
			insertObjID, ok := id.(primitive.ObjectID)
			// if _id is generated by mongo
			if ok {
				res = append(res, insertObjID.Hex())
				continue
			}
			res = append(res, id)
		}
	}
	if e != nil {
		/*
			if ordered is true, you will be able to identify which insert failed and which succeeded since all
			subsequent inserts after a failure will not be executed.
			if ordered is false, you will not be able to identify which insert failed and which succeeded if default _id
			is used since they will be executed in parallel. If _id is custom, you will be able to identify which insert
			failed.
		*/
		client.log.Error("MONGO_INSERTMANY", e)
		return 0, res, e
	}
	return len(res), res, e
}

// InsertOne - function to insert a single doc into collection, ctx can be nil
func (client Client) InsertOne(ctx context.Context, colname string, doc interface{}) (interface{}, error) {
	// select collection
	col := client.Db.Collection(colname)

	// insert
	result, e := col.InsertOne(ctx, doc)
	if e != nil {
		client.log.Error("MONGO_INSERTONE", e)
		return "", e
	}

	insertObjID, ok := result.InsertedID.(primitive.ObjectID)
	// if _id is generated by mongo
	if ok {
		return insertObjID.Hex(), e
	}
	/*
		_id can be different data types(e.g string, int, etc) since this can be overridden by client,
		so we need to return interface
	*/
	return result.InsertedID, e
}

// UpdateMany - function to update many docs in the collection, ctx can be nil
func (client Client) UpdateMany(ctx context.Context, colname string, filter interface{},
	update interface{}) (int64, error) {
	// select collection
	col := client.Db.Collection(colname)

	// update
	result, e := col.UpdateMany(ctx, filter, update)
	if e != nil {
		client.log.Error("MONGO_UPDATEMANY", e)
		return 0, e
	}
	return result.ModifiedCount, e
}

// UpdateOne - function to update a single doc in the collection, ctx can be nil
func (client Client) UpdateOne(ctx context.Context, colname string,
	filter interface{}, update interface{}, opts *options.UpdateOptions) (int64, error) {
	// select collection
	col := client.Db.Collection(colname)

	// update
	result, e := col.UpdateOne(ctx, filter, update, opts)
	if e != nil {
		client.log.Error("MONGO_UPDATEONE", e)
		return 0, e
	}
	return result.ModifiedCount, e
}

// DeleteMany - function to delete many docs in the collection, ctx can be nil
func (client Client) DeleteMany(ctx context.Context, colname string, filter interface{}) (int64, error) {
	// select collection
	col := client.Db.Collection(colname)

	// delete
	result, e := col.DeleteMany(ctx, filter)
	if e != nil {
		client.log.Error("MONGO_DELETEMANY", e)
		return 0, e
	}
	return result.DeletedCount, e
}

// DeleteOne - function to delete a single doc in the collection, ctx can be nil
func (client Client) DeleteOne(ctx context.Context, colname string, filter interface{}) (int64, error) {
	// select collection
	col := client.Db.Collection(colname)

	// delete
	result, e := col.DeleteOne(ctx, filter)
	if e != nil {
		client.log.Error("MONGO_DELETEONE", e)
		return 0, e
	}
	return result.DeletedCount, e
}

// ReplaceOne - function to replace a single doc in the collection, ctx can be nil
func (client Client) ReplaceOne(ctx context.Context, colname string,
	filter interface{}, replace interface{}, opts *options.ReplaceOptions) (int64, error) {
	// select collection
	col := client.Db.Collection(colname)

	// replace
	result, e := col.ReplaceOne(ctx, filter, replace, opts)
	if e != nil {
		client.log.Error("MONGO_REPLACEONE", e)
		return 0, e
	}
	return result.ModifiedCount, e
}

// BulkWrite - function to write on multiple documents
// ctx can be nil, models are an array of operations to execute on the collection, opts is optional (allow to be not provided without breaking function in Go)
func (client Client) BulkWrite(ctx context.Context, collname string, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (mongo.BulkWriteResult, error) {

	// select the collection
	coll := client.Db.Collection(collname)

	result, err := coll.BulkWrite(ctx, models, opts...)
	if err != nil {
		client.log.Error("MONGO_BULKWRITE", err)
		return mongo.BulkWriteResult{}, err
		// we dont return an int as operations performed information is useful (i.e. modified count, upsert count, etc, upserted IDs)
	}

	return *result, nil
}
