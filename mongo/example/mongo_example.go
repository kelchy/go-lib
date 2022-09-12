package main

import (
	"fmt"
	"github.com/kelchy/go-lib/mongo"
)

const (
	/*
		replica set needed to test transaction. the minimum mongodb version required is 4.0.
		guide on how to set up a replica set locally:
		https://medium.com/@OndrejKvasnovsky/mongodb-replica-set-on-local-macos-f5fc383b3fd6
	 */
	mongoURI = "mongodb://localhost:27017,127.0.0.1:27018/test-db?replicaSet=rs0"
	collectionName = "test-collection"
)

func main() {
	mongoClient, err := mongo.New(mongoURI, 10)
	if err!= nil {
		fmt.Println("Error connecting to mongo: ", err)
		return
	}

	findExamples(&mongoClient)
	insertOneExamples(&mongoClient)
	insertManyExamples(&mongoClient)
	updateOneExamples(&mongoClient)
	updateManyExamples(&mongoClient)
	deleteOneExamples(&mongoClient)
	deleteManyExamples(&mongoClient)
	transactionExamples(&mongoClient)
}

func findExamples(mongoClient *mongo.Client) {
	filter := map[string]interface{}{
		"product": "PRD1",
	}
	result, err := mongoClient.Find(nil, collectionName, filter, nil)
	fmt.Println(fmt.Sprintf("find. result: %+v err: %+v", result, err))
}

func insertOneExamples(mongoClient *mongo.Client) {
	// insert without _id
	doc := map[string]interface{}{
		"product": "PRD1",
		"price": 5,
	}
	id, err := mongoClient.InsertOne(nil, collectionName, doc)
	fmt.Println(fmt.Sprintf("insert one without _id. id: %+v err: %+v", id, err))

	// insert with _id
	doc = map[string]interface{}{
		"_id": 12345,
		"product": "PRD2",
		"price": 5,
	}
	id, err = mongoClient.InsertOne(nil, collectionName, doc)
	fmt.Println(fmt.Sprintf("insert one with _id. id: %+v err: %+v", id, err))
}

func insertManyExamples(mongoClient *mongo.Client) {
	insertList := []interface{}{
		map[string]interface{}{
			"product": "PRD3",
			"price": 10,
		},
		map[string]interface{}{
			"product": "PRD4",
			"price": 15,
		},
		map[string]interface{}{
			"product": "PRD5",
			"price": 20,
		},
		map[string]interface{}{
			"product": "PRD5",
			"price": 25,
		},
		map[string]interface{}{
			"product": "PRD5",
			"price": 25,
		},
	}

	count, ids, err := mongoClient.InsertMany(nil, collectionName, insertList, false)
	fmt.Println(fmt.Sprintf("insert many. count: %+v ids: %+v err: %+v", count, ids, err))
}

func updateOneExamples(mongoClient *mongo.Client) {
	filter := map[string]interface{}{
		"product": "PRD5",
	}
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"price": 100,
		},
	}

	count, err := mongoClient.UpdateOne(nil, collectionName, filter, update, nil)
	fmt.Println(fmt.Sprintf("update one. count: %+v err: %+v", count, err))
}

func updateManyExamples(mongoClient *mongo.Client) {
	filter := map[string]interface{}{
		"product": "PRD5",
	}
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"price": 200,
		},
	}

	count, err := mongoClient.UpdateMany(nil, collectionName, filter, update)
	fmt.Println(fmt.Sprintf("update many. count: %+v err: %+v", count, err))
}

func deleteOneExamples(mongoClient *mongo.Client) {
	filter := map[string]interface{}{
		"product": "PRD5",
	}

	count, err := mongoClient.DeleteOne(nil, collectionName, filter)
	fmt.Println(fmt.Sprintf("delete one. count: %+v err: %+v", count, err))
}

func deleteManyExamples(mongoClient *mongo.Client) {
	filter := map[string]interface{}{
		"product": "PRD5",
	}

	count, err := mongoClient.DeleteMany(nil, collectionName, filter)
	fmt.Println(fmt.Sprintf("delete many. count: %+v err: %+v", count, err))
}

func transactionExamples(mongoClient *mongo.Client) {
	actions := []map[string]interface{}{
		{
			"operation": "insertOne",
			"collection": collectionName,
			"doc": map[string]interface{}{
				"_id": 12345,
				"product": "PRD2",
				"price": 5,
			},
		},
		{
			"operation": "insertMany",
			"collection": collectionName,
			"docs": []interface{}{
				map[string]interface{}{
					"product": "PRD3",
					"price": 10,
				},
				map[string]interface{}{
					"product": "PRD4",
					"price": 15,
				},
				map[string]interface{}{
					"product": "PRD5",
					"price": 20,
				},
				map[string]interface{}{
					"product": "PRD5",
					"price": 25,
				},
				map[string]interface{}{
					"product": "PRD6",
					"price": 25,
				},
			},
			"ordered": true,
		},
		{
			"operation": "updateOne",
			"collection": collectionName,
			"filter": map[string]interface{}{
				"product": "PRD6",
			},
			"update" : map[string]interface{}{
				"$set": map[string]interface{}{
					"price": 100,
				},
			},
		},
		{
			"operation": "updateMany",
			"collection": collectionName,
			"filter": map[string]interface{}{
				"product": "PRD5",
			},
			"update": map[string]interface{}{
				"$set": map[string]interface{}{
					"price": 200,
				},
			},
		},
	}

	response, err := mongoClient.Transaction(actions)
	fmt.Println(fmt.Sprintf("transaction. response: %+v err: %+v", response, err))

	actions = append(actions, map[string]interface{}{
			"operation": "invalidOperation",
			"collection": collectionName,
			"filter": map[string]interface{}{
				"product": "PRD5",
			},
			"update": map[string]interface{}{
				"$set": map[string]interface{}{
					"price": 200,
				},
			},
	})
	response, err = mongoClient.Transaction(actions)
	fmt.Println(fmt.Sprintf("transaction. response: %+v err: %+v", response, err))
}
