package mongo

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// so that we don't need to import bson/primitive everywhere
// go 1.11 does not have this problem as it allows us to use the base type
// we have to add here as needed by devs
// alternatively, we also export these primitives for convenience

// DateTime - re-export primitive.DateTime
var DateTime primitive.DateTime

// M - re-export primitive.M
var M primitive.M

// A - re-export primitive.A
var A primitive.A

// IntDateTime - the purpose is to convert a primitive.DateTime to int64
func IntDateTime(i interface{}) int64 {
	return int64(i.(primitive.DateTime))
}

// MapInterface - the purpose is to convert a primitive.M to map[string]interface{}
func MapInterface(i interface{}) map[string]interface{} {
	return map[string]interface{}(i.(primitive.M))
}

// SliceInterface - the purpose is to convert a primitive.A to []interface{}
func SliceInterface(i interface{}) []interface{} {
	return []interface{}(i.(primitive.A))
}
