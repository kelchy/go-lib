package rmq

import (
	"github.com/google/uuid"
)

// newUUID generates a random UUID according to RFC 4122
func NewUUID() (string, error) {
	id, err := uuid.NewRandom() // returns id and err without panic
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
