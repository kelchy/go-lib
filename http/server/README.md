```
	// initialize with empty cors setting
        rtr := server.New([]string{})

	// sample custom middleware
        rtr.Use(func (next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        next.ServeHTTP(w, r)
                })
        })

	// use "esc" to create a "codified" version of static files and declare like this
	// https://github.com/mjibson/esc
	// for example ~/go/bin/esc -o testdir/test.go -pkg static -ignore=".*.go" testdir
        rtr.StaticFs("/test/", static.FS(false))

	// api definition
        rtr.Get("/welcome", func(w http.ResponseWriter, r *http.Request) {
                server.JSON(w, r, map[string]string{
                        "status": "success",
                })
        })

	// run server with cleartext http/2
        rtr.Run("h2c", ":8080")
```

### Graceful shutdown

`Run` and `RunS` stop on SIGINT or SIGTERM and shut down gracefully: in-flight requests get up
to 10 seconds to finish, and the call returns nil. Use `RunContext` / `RunSContext` to stop on
your own context instead.

```
	// on stop, report draining, wait 5s for the load balancer to stop sending, then give
	// in-flight requests up to 20s
	rtr.SetShutdown(20*time.Second, 5*time.Second)

	rtr.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if rtr.Draining() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	rtr.Run("http", ":8080")
```

### Timeouts

Every server gets connection timeouts by default (`DefaultTimeouts`): 5s to read the request
headers, 10s to read the request, 10s to write the response and 60s for idle keep-alive
connections. Change them with `SetTimeouts`, for example for a streaming endpoint; a 0 field
means no timeout.

```
	rtr.SetTimeouts(server.Timeouts{ReadHeader: 5 * time.Second, Read: 10 * time.Second, Write: 2 * time.Minute, Idle: 60 * time.Second})
```
