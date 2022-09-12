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
