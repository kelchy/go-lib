package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/urfave/negroni"
)

func (rtr *Router) catchall(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		w2 := negroni.NewResponseWriter(w)
		// defer is first in last out, this will run if in case any
		// uncaught panic happens within the api logic, except if
		// it happens within another go routine created within
		defer func() {
			rc := recover()
			diff := float64(time.Since(t1).Microseconds()) / 1000
			diffStr := fmt.Sprintf("%f", diff)
			if rc != nil {
				rtr.log.Error("HTTPS_MW", errors.New("Uncaught Exception: "+rc.(error).Error()))
				// build generic 500 error
				jsonBody, _ := json.Marshal(map[string]string{
					"error": "There was an internal server error",
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(jsonBody)
			} else if rtr.logRequest {
				msg, _ := json.Marshal(map[string]string{
					"method": r.Method,
					"status": strconv.Itoa(w2.Status()),
					"src":    r.RemoteAddr,
					"ms":     diffStr,
				})
				rtr.log.Out(r.URL.Path, string(msg))
			}
		}()
		next.ServeHTTP(w2, r)
	})
}
