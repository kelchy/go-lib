package main

import (
	"errors"
	Log "github.com/kelchy/go-lib/log"
)

func main() {
	log, err := Log.New("standard")	// or empty string
	if err != nil {
		panic(err)
	}
	log.Out("Example: scope", "message")
	log.Debug("Example: scope", "you should not see this if GO_ENV is production")

	empty, _ := Log.New("empty")
	empty.Out("Empty: You should", "not see this")
	empty.Error("Empty", errors.New("You should not see this"))

	erroronly, _ := Log.New("erroronly")
	// by default, would be in json
	erroronly.Out("Erroronly: You should", "not see this")
	erroronly.Error("Erroronly", errors.New("You should see this"))

	// turn off json logging
	erroronly.JSONDisable()
	erroronly.Error("Erroronly", errors.New("You should not see this as json"))
}
