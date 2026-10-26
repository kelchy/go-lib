package server

import (
	"net/http"
	"time"
)

// Timeouts - the server's connection timeouts (see net/http.Server). A zero value
// means no timeout, as in net/http
type Timeouts struct {
	ReadHeader time.Duration
	Read       time.Duration
	Write      time.Duration
	Idle       time.Duration
}

// DefaultTimeouts - the timeouts every server gets unless SetTimeouts changes them
var DefaultTimeouts = Timeouts{
	ReadHeader: 5 * time.Second,
	Read:       10 * time.Second,
	Write:      10 * time.Second,
	Idle:       60 * time.Second,
}

// SetTimeouts - replaces the server's connection timeouts, for example a longer Write
// for streaming responses. Set a field to 0 for no timeout
func (rtr *Router) SetTimeouts(t Timeouts) {
	rtr.timeouts = &t
}

// applyTimeouts - sets the router's timeouts, or the defaults, on srv
func (rtr Router) applyTimeouts(srv *http.Server) {
	t := DefaultTimeouts
	if rtr.timeouts != nil {
		t = *rtr.timeouts
	}
	srv.ReadHeaderTimeout = t.ReadHeader
	srv.ReadTimeout = t.Read
	srv.WriteTimeout = t.Write
	srv.IdleTimeout = t.Idle
}
