// Package auth provides reusable identity verification and token primitives.
// Applications own authorization, HTTP handlers, sessions, and persistence.
package auth

import (
	"crypto/rand"
	"encoding/base64"
)

var readRandom = rand.Read

func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := readRandom(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
