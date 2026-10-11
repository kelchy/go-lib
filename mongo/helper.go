// Package mongo provides helpers for MongoDB driver results.
package mongo

import drivermongo "go.mongodb.org/mongo-driver/v2/mongo"

// SingleResult wraps the MongoDB driver's single-document result.
type SingleResult struct {
	result *drivermongo.SingleResult
}

// NewSingleResult wraps a driver result.
func NewSingleResult(result *drivermongo.SingleResult) *SingleResult {
	return &SingleResult{result: result}
}

// Decode decodes the result into val.
func (result *SingleResult) Decode(val any) error {
	return result.result.Decode(val)
}
