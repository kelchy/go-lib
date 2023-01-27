package server

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/kelchy/go-lib/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// ChiRouter - interface for chi router
type ChiRouter = chi.Router

// Router - initialized instance
type Router struct {
	Engine     *chi.Mux
	log        log.Log
	logRequest bool
}

// New - constructor function to initialize instance
func New(origins []string, headers []string) (*Router, error) {
	var rtr Router

	l, e := log.New("")
	if e != nil {
		return &rtr, e
	}
	rtr.log = l
	rtr.logRequest = true

	if len(origins) == 0 {
		origins = []string{"http://localhost", "https://localhost"}
	}
	allowedDefault := []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}
	if len(headers) == 0 {
		headers = allowedDefault
	} else {
		headers = append(headers, allowedDefault...)
	}
	rtr.Engine = chi.NewRouter()
	rtr.Engine.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{ "GET", "POST", "PUT", "DELETE", "OPTIONS" },
		AllowedHeaders:   headers,
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60,
	}))
	rtr.Engine.Use(middleware.RealIP)
	rtr.Engine.Use(rtr.catchall)
	return &rtr, nil
}

// SetLogger - changes the logger mode
func (rtr *Router) SetLogger(logtype string) {
	l, e := log.New(logtype)
	if e == nil {
		rtr.log = l
	}
}

// SetLogRequest - changes behaviour on whether to log requests or not
func (rtr *Router) SetLogRequest(lr bool) {
	rtr.logRequest = lr
}

// Run - run and listen for http
func (rtr Router) Run(proto string, hostport string) error {
	rtr.log.Out("SERVER_RUN", "Listening "+proto+" "+hostport)
	var e error
	if proto == "http" {
		e = http.ListenAndServe(hostport, rtr.Engine)
	} else if proto == "h2c" {
		// h2c denotes http/2 in cleartext, useful in cases where API GW strips encryption
		h2s := &http2.Server{}
		http.ListenAndServe(hostport, h2c.NewHandler(rtr.Engine, h2s))
	} else {
		e = errors.New("Unknown Proto")
	}
	if e != nil {
		rtr.log.Error("SERVER_RUN", e)
	}
	return e
}

// RunS - run and listen for https
func (rtr Router) RunS(proto string, hostport string, crt string, key string) error {
	rtr.log.Out("SERVER_RUNS", "Listening "+proto+" "+hostport)
	var e error
	if proto == "https" {
		e = http.ListenAndServeTLS(hostport, crt, key, rtr.Engine)
	} else if proto == "h2" {
		// TODO: add h2
	} else {
		e = errors.New("Unknown Proto")
	}
	if e != nil {
		rtr.log.Error("SERVER_RUNS", e)
	}
	return e
}

// Static - function to handle and serve static files within a directory on live system
func (rtr Router) Static(urlPath string, dirPath string) {
	// do not use wildcard (*) in urlPath
	rtr.Engine.Handle(urlPath+"*", http.StripPrefix(urlPath, http.FileServer(http.Dir(dirPath))))
	// TODO: handle wildcards better
}

// StaticFs - function to handle and serve static files embedded into binary using esc
func (rtr Router) StaticFs(urlPath string, fs http.FileSystem) {
	/*
		generate a boxed static filesystem by using esc (https://github.com/mjibson/esc):
			~/go/bin/esc -o static/static.go -pkg static -ignore=".*.go" ./static
		on your project's home dir, use it like this:
			var rtr server.Router
			rtr.New([]string{})
			rtr.StaticFs("/static/", static.FS(false))
		static.FS() returns a standard http.FileSystem which you can pass to this function
	*/
	// do not use wildcard (*) in urlPath
	rtr.Engine.Handle(urlPath+"*", http.FileServer(fs))
	// TODO: handle wildcards better
}
