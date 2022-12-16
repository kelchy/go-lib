package main

import (
	"errors"
	"net/http"

	"github.com/kelchy/go-lib/http/server"
)

func main() {
	// initialize with empty cors setting
	rtr, _ := server.New([]string{"http://localhost:8080"})

	// Change logger to log different levels.
	// Available levels: "empty", "erroronly", "Standard"
	rtr.SetLogger("erroronly")

	// Changes logger to log requests or not
	rtr.SetLogRequest(true)

	// Add a catchall middleware to log requests and handle errors
	// This is optional, but recommended
	// SetLogRequest and SetLogger must be called before this for the correct behaviour
	rtr.AddCatchAll()

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

	// run server with http
	rtr.Run("http", ":8080")
}
