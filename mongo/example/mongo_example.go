package main

import (
	"context"
	"fmt"
	"github.com/kelchy/go-lib/mongo"
)

const (
	/*
		replica set needed to test transaction. the minimum mongodb version required is 4.0.
		guide on how to set up a replica set locally:
		https://medium.com/@OndrejKvasnovsky/mongodb-replica-set-on-local-macos-f5fc383b3fd6
	*/
	mongoURI       = "mongodb://localhost:27017,127.0.0.1:27018/test-db?replicaSet=rs0"
	collectionName = "test-collection"
)

func main() {
	mongoClient, err := mongo.New(mongoURI)
	if err != nil {
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
	result, err := mongoClient.Find(context.TODO(), collectionName, filter, nil)
	fmt.Printf("find. result: %+v err: %+v\n", result, err)
}

func insertOneExamples(mongoClient *mongo.Client) {
	// insert without _id
	doc := map[string]interface{}{
		"product": "PRD1",
		"price":   5,
	}
	id, err := mongoClient.InsertOne(context.TODO(), collectionName, doc)
	fmt.Printf("insert one without _id. id: %+v err: %+v\n", id, err)

	// insert with _id
	doc = map[string]interface{}{
		"_id":     12345,
		"product": "PRD2",
		"price":   5,
	}
	id, err = mongoClient.InsertOne(context.TODO(), collectionName, doc)
	fmt.Printf("insert one with _id. id: %+v err: %+v\n", id, err)
}

func insertManyExamples(mongoClient *mongo.Client) {
	insertList := []interface{}{
		map[string]interface{}{
			"product": "PRD3",
			"price":   10,
		},
		map[string]interface{}{
			"product": "PRD4",
			"price":   15,
		},
		map[string]interface{}{
			"product": "PRD5",
			"price":   20,
		},
		map[string]interface{}{
			"product": "PRD5",
			"price":   25,
		},
		map[string]interface{}{
			"product": "PRD5",
			"price":   25,
		},
	}

	count, ids, err := mongoClient.InsertMany(context.TODO(), collectionName, insertList, false)
	fmt.Printf("insert many. count: %+v ids: %+v err: %+v\n", count, ids, err)
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

	count, err := mongoClient.UpdateOne(context.TODO(), collectionName, filter, update, nil)
	fmt.Printf("update one. count: %+v err: %+v\n", count, err)
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

	count, err := mongoClient.UpdateMany(context.TODO(), collectionName, filter, update)
	fmt.Printf("update many. count: %+v err: %+v\n", count, err)
}

func deleteOneExamples(mongoClient *mongo.Client) {
	filter := map[string]interface{}{
		"product": "PRD5",
	}

	count, err := mongoClient.DeleteOne(context.TODO(), collectionName, filter)
	fmt.Printf("delete one. count: %+v err: %+v\n", count, err)
}

func deleteManyExamples(mongoClient *mongo.Client) {
	filter := map[string]interface{}{
		"product": "PRD5",
	}

	count, err := mongoClient.DeleteMany(context.TODO(), collectionName, filter)
	fmt.Printf("delete many. count: %+v err: %+v\n", count, err)
}

func transactionExamples(mongoClient *mongo.Client) {
	actions := []map[string]interface{}{
		{
			"operation":  "insertOne",
			"collection": collectionName,
			"doc": map[string]interface{}{
				"_id":     12345,
				"product": "PRD2",
				"price":   5,
			},
		},
		{
			"operation":  "insertMany",
			"collection": collectionName,
			"docs": []interface{}{
				map[string]interface{}{
					"product": "PRD3",
					"price":   10,
				},
				map[string]interface{}{
					"product": "PRD4",
					"price":   15,
				},
				map[string]interface{}{
					"product": "PRD5",
					"price":   20,
				},
				map[string]interface{}{
					"product": "PRD5",
					"price":   25,
				},
				map[string]interface{}{
					"product": "PRD6",
					"price":   25,
				},
			},
			"ordered": true,
		},
		{
			"operation":  "updateOne",
			"collection": collectionName,
			"filter": map[string]interface{}{
				"product": "PRD6",
			},
			"update": map[string]interface{}{
				"$set": map[string]interface{}{
					"price": 100,
				},
			},
		},
		{
			"operation":  "updateMany",
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

	response, err := mongoClient.Transaction(actions, 10)
	fmt.Printf("transaction. response: %+v err: %+v\n", response, err)

	actions = append(actions, map[string]interface{}{
		"operation":  "invalidOperation",
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
	response, err = mongoClient.Transaction(actions, 10)
	fmt.Printf("transaction. response: %+v err: %+v\n", response, err)
}
