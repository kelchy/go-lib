package server

import (
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
	Engine      *chi.Mux
	log         log.Log
	logRequest  bool
	logSkipPath []string
	shutdown    *shutdown
}

// CorsOptions takes in the options for CORS
type CorsOptions struct {
	Origins           []string
	Headers           []string
	AllowedOriginFunc func(r *http.Request, origin string) bool
}

// New - constructor function to initialize an instance
func New(origins []string, headers []string) (*Router, error) {
	corsOptions := &CorsOptions{
		Origins: origins,
		Headers: headers,
	}

	return NewWithCorsOptions(corsOptions)
}

// NewWithCorsOptions - constructor function to initialize with cors options
func NewWithCorsOptions(corsOptions *CorsOptions) (*Router, error) {
	var rtr Router

	l, e := log.New("")
	if e != nil {
		return &rtr, e
	}
	rtr.log = l
	rtr.logRequest = true
	rtr.shutdown = &shutdown{timeout: DefaultShutdownTimeout}

	// by default middleware don't log root path which is
	// usually used by health checks
	rtr.logSkipPath = []string{"/"}
	origins := corsOptions.Origins
	if len(origins) == 0 {
		origins = []string{"http://localhost", "https://localhost"}
	}
	allowedDefault := []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}
	headers := corsOptions.Headers
	if len(headers) == 0 {
		headers = allowedDefault
	} else {
		headers = append(headers, allowedDefault...)
	}
	rtr.Engine = chi.NewRouter()
	rtr.Engine.Use(cors.Handler(cors.Options{
		AllowOriginFunc:  corsOptions.AllowedOriginFunc,
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
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

// SetLogSkipPath - changes the middleware logging behaviour
func (rtr *Router) SetLogSkipPath(list []string) {
	rtr.logSkipPath = list
}

// SetLogRequest - changes behaviour on whether to log requests or not
func (rtr *Router) SetLogRequest(lr bool) {
	rtr.logRequest = lr
}

// Run - run and listen for http until SIGINT or SIGTERM, then shut down gracefully
// (see SetShutdown). Returns nil after a graceful shutdown
func (rtr Router) Run(proto string, hostport string) error {
	ctx, stop := signalContext()
	defer stop()
	return rtr.RunContext(ctx, proto, hostport)
}

// RunS - run and listen for https until SIGINT or SIGTERM, then shut down gracefully
// (see SetShutdown). Returns nil after a graceful shutdown
func (rtr Router) RunS(proto string, hostport string, crt string, key string) error {
	ctx, stop := signalContext()
	defer stop()
	return rtr.RunSContext(ctx, proto, hostport, crt, key)
}

// h2cHandler - serves http/2 in cleartext as well as http/1
func h2cHandler(h http.Handler) http.Handler {
	return h2c.NewHandler(h, &http2.Server{})
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
