package main

import (
	"errors"
	"net/http"

	"github.com/kelchy/go-lib/http/server"
)

func main() {
	// initialize with empty cors setting
        rtr, _ := server.New([]string{"http://localhost:8080"}, []string{"X-CUSTOM-HEADER"})

	// sample custom middleware
	rtr.Engine.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})

	// use "esc" to create a "codified" version of static files and declare like this
	// https://github.com/mjibson/esc
	// for example ~/go/bin/esc -o testdir/test.go -pkg static -ignore=".*.go" testdir
	//rtr.StaticFs("/test/", static.FS(false))

	// api definition
	rtr.Get("/welcome", func(w http.ResponseWriter, r *http.Request) {
		server.JSON(w, r, map[string]string{
			"status": "success",
		})
	})
	rtr.Get("/crash", func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("deliberate crash"))
	})

	// Disable automatic logging
	rtr.SetLogRequest(false)

	// run server with cleartext http/2
	rtr.Run("http", ":8080")
}
