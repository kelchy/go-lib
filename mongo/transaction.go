package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kelchy/go-lib/common"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
	***** Transaction is only supported on mongo 4.0 and above *****
	generic function to perform transaction. this function should support all kind of operations(update, insert, delete,
	etc) and arbitrary number of times in specified order. the argument is an array and will be executed in order.
	each action will be a map and depending on the action, there will be key-value pairs that is only required by that action.
	standard keys:
		1. operation: the name of the action such as updateOne etc. List of operations is in the constant file
		2. collection: the collection to perform the action
	remaining keys will follow the namings of the arguments of the respective method. for exmaple:
		to perform update one, the params are:
		UpdateOne(sessCtx context.Context, timeout int, colname string, filter interface{}, update interface{})
		so we should pass below object where the value for 'filter' in the object below will be the 'filter' argument in
		the above method and the value for 'update' in the object below will be the 'update' argument in the above method
		{
			"operation": updateOne,
			"collection": CollectionName,
			"filter": map[string]interface{}{
				"product": "PRD6",
			},
			"update" : map[string]interface{}{
				"$set": map[string]interface{}{
					"price": 100,
				},
			},
		},

	as of now, only these operations are supported:
		1. updateOne
		2. insertOne
		3. insertMany
		4. updateMany
*/

// Transaction - creates a transaction
func (client Client) Transaction(actions []map[string]interface{}, timeout int) (interface{}, error) {
	// validate operations first
	for _, action := range actions {
		if !common.SliceHasString(Operations, action["operation"].(string)) {
			return nil, errors.New("Unknown operation - " + action["operation"].(string))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	sess, err := client.Connection.StartSession()
	if err != nil {
		client.log.Error("MONGO_TRANSACTION", err)
		return nil, err
	}
	defer sess.EndSession(ctx)

	result := []interface{}{}

	err = mongo.WithSession(ctx, sess, func(sessCtx mongo.SessionContext) error {
		if err := sess.StartTransaction(); err != nil {
			return err
		}

		for _, action := range actions {
			collection, _ := action["collection"].(string)
			switch action["operation"] {
			case "updateOne":
				// keys used in action object: filter, update
				count, err := client.UpdateOne(sessCtx, collection, action["filter"], action["update"], nil)
				if err != nil {
					client.log.Error("MONGO_TRANSACTION_UPDATEONE", fmt.Errorf("action: %+v. err: %+v", action, err))
					// Use context.Background() to ensure that the commit can complete successfully even if the context passed to
					// mongo.WithSession is changed to have a timeout.
					_ = sess.AbortTransaction(context.Background())
					return err
				}
				result = append(result, map[string]interface{}{
					"operation":    "updateOne",
					"updatedCount": count,
				})
			case "updateMany":
				// keys used in action object: filter, update
				count, err := client.UpdateMany(sessCtx, collection, action["filter"], action["update"])
				if err != nil {
					client.log.Error("MONGO_TRANSACTION_UPDATEMANY", fmt.Errorf("action: %+v. err: %+v", action, err))
					// Use context.Background() to ensure that the commit can complete successfully even if the context passed to
					// mongo.WithSession is changed to have a timeout.
					_ = sess.AbortTransaction(context.Background())
					return err
				}
				result = append(result, map[string]interface{}{
					"operation":    "updateMany",
					"updatedCount": count,
				})
			case "insertOne":
				// keys used in action object: doc
				id, err := client.InsertOne(sessCtx, collection, action["doc"])
				if err != nil {
					client.log.Error("MONGO_TRANSACTION_INSERTONE", fmt.Errorf("action: %+v. err: %+v", action, err))
					// Use context.Background() to ensure that the commit can complete successfully even if the context passed to
					// mongo.WithSession is changed to have a timeout.
					_ = sess.AbortTransaction(context.Background())
					return err
				}
				result = append(result, map[string]interface{}{
					"operation":  "insertOne",
					"insertedId": id,
				})
			case "insertMany":
				// keys used in action object: ordered, docs
				ordered, _ := action["ordered"].(bool)
				docs, _ := action["docs"].([]interface{})
				count, ids, err := client.InsertMany(sessCtx, collection, docs, ordered)
				if err != nil {
					client.log.Error("MONGO_TRANSACTION_INSERTMANY", fmt.Errorf("action: %+v. err: %+v", action, err))
					// Use context.Background() to ensure that the commit can complete successfully even if the context passed to
					// mongo.WithSession is changed to have a timeout.
					_ = sess.AbortTransaction(context.Background())
					return err
				}
				result = append(result, map[string]interface{}{
					"operation":     "insertMany",
					"insertedCount": count,
					"insertIds":     ids,
				})
			}
		}

		// Use context.Background() to ensure that the commit can complete successfully even if the context passed to
		// mongo.WithSession is changed to have a timeout.
		return sess.CommitTransaction(context.Background())
	})

	return result, err
}
