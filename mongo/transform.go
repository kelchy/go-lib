package mongo

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// so that we don't need to import v2/bson everywhere
// go 1.11 does not have this problem as it allows us to use the base type
// we have to add here as needed by devs

// DateTime - re-export bson.DateTime
var DateTime bson.DateTime

// M - re-export bson.M
var M bson.M

// A - re-export bson.A
var A bson.A

// IntDateTime - the purpose is to convert a bson.DateTime to int64
func IntDateTime(i interface{}) int64 {
	return int64(i.(bson.DateTime))
}

// MapInterface - the purpose is to convert a bson.M to map[string]interface{}
func MapInterface(i interface{}) map[string]interface{} {
	return map[string]interface{}(i.(bson.M))
}

// SliceInterface - the purpose is to convert a bson.A to []interface{}
func SliceInterface(i interface{}) []interface{} {
	return []interface{}(i.(bson.A))
}
