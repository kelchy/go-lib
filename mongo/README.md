# MongoDB Wrapper
- https://www.mongodb.com
- https://github.com/mongodb/mongo-go-driver

## Usage
```
      import (
              "github.com/kelchy/go-lib/mongo"
      )
```
- Initialization - for simplicity this example shows getting from env. but developer can opt to use config manager
```
uri := os.Getenv("MONGOURI")
Mongo, e := mongo.New(uri)
```
- Insert doc in collection
```
        list := []interface{}{}
        inserted := Mongo.InsertMany(nil, 0, "cdr", list, false)
```
- Find doc in collection
```
        cdrs, e := Mongo.Find(nil, 0, "cdr", mongo.M{}, mongo.M{"sort": "start", "order": int64(-1), "skip": int64(0), "limit": int64(100)})
        if e != nil {
                res.Error = e.Error()
                router.JSON(w, r, res)
        }
```
- Insert file in gridFS
```
        filesize, e := Mongo.FSset("/tmp/cdr.csv")
        if e != nil {
                log.Error(e)
        }
```
- Find file in gridFS
```
        files, e := Mongo.FSlist(mongo.M{"metadata.type": listType}, "uploadDate", -1, 0, 100)
        if e != nil {
                log.Error(e)
                router.JSON(w, r, res)
                return
        }
```

- Perform transaction for arbitrary number of operations and different types of operations
```
    actions := []map[string]interface{}{
        {
            "operation": "insertOne", // the type of operation. currently only updateOne, insertOne, insertMany, updateMany are supported
            "collection": "test", // collection name
            "doc": map[string]interface{}{
                "_id": 12345,
                "product": "PRD2",
                "price": 5,
            },
        },
        {
            "operation": "updateOne",
            "collection": "test",
			"filter": map[string]interface{}{
				"product": "PRD6",
			},
			"update" : map[string]interface{}{
				"$set": map[string]interface{}{
					"price": 100,
				},
			},
        },
    }
   response, err := mongoClient.Transaction(actions)
```
