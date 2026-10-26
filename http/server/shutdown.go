package server

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

// DefaultShutdownTimeout - how long a shutdown waits for in-flight requests by default
const DefaultShutdownTimeout = 10 * time.Second

// shutdown - graceful shutdown settings, shared by copies of a Router
type shutdown struct {
	timeout  time.Duration
	delay    time.Duration
	draining int32
}

// listen - opens the server's listener, replaced in tests
var listen = net.Listen

// signalContext - the context Run and RunS stop on, replaced in tests
var signalContext = func() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// SetShutdown - changes how the server shuts down. Once asked to stop, the server
// reports Draining, waits delay so load balancers stop sending new requests, then
// gives in-flight requests up to timeout to finish
func (rtr *Router) SetShutdown(timeout time.Duration, delay time.Duration) {
	sd := rtr.shutdownSettings()
	sd.timeout = timeout
	sd.delay = delay
}

// Draining - true once the server has started shutting down, for readiness checks
func (rtr Router) Draining() bool {
	return rtr.shutdown != nil && atomic.LoadInt32(&rtr.shutdown.draining) == 1
}

// RunContext - run and listen for http until ctx is done, then shut down gracefully.
// Returns nil after a graceful shutdown
func (rtr Router) RunContext(ctx context.Context, proto string, hostport string) error {
	rtr.log.Out("SERVER_RUN", "Listening "+proto+" "+hostport)
	var e error
	switch proto {
	case "http":
		e = rtr.serve(ctx, &http.Server{Addr: hostport, Handler: rtr.Engine}, nil)
	case "h2c":
		// h2c denotes http/2 in cleartext, useful in cases where API GW strips encryption
		e = rtr.serve(ctx, &http.Server{Addr: hostport, Handler: h2cHandler(rtr.Engine)}, nil)
	default:
		e = errors.New("unknown proto")
	}
	if e != nil {
		rtr.log.Error("SERVER_RUN", e)
	}
	return e
}

// RunSContext - run and listen for https until ctx is done, then shut down gracefully.
// Returns nil after a graceful shutdown
func (rtr Router) RunSContext(ctx context.Context, proto string, hostport string, crt string, key string) error {
	rtr.log.Out("SERVER_RUNS", "Listening "+proto+" "+hostport)
	var e error
	switch proto {
	case "https":
		e = rtr.serve(ctx, &http.Server{Addr: hostport, Handler: rtr.Engine}, []string{crt, key})
	case "h2":
		// TODO: add h2
	default:
		e = errors.New("unknown proto")
	}
	if e != nil {
		rtr.log.Error("SERVER_RUNS", e)
	}
	return e
}

// serve - listens until ctx is done, then drains: Draining turns true, the server
// waits the shutdown delay and gives in-flight requests the shutdown timeout. A bad
// address or certificate fails straight away, before anything is served. certKey is
// the certificate and key files for https, nil for http
func (rtr Router) serve(ctx context.Context, srv *http.Server, certKey []string) error {
	sd := rtr.shutdown
	if sd == nil {
		sd = &shutdown{timeout: DefaultShutdownTimeout}
	}
	if certKey != nil {
		cert, e := tls.LoadX509KeyPair(certKey[0], certKey[1])
		if e != nil {
			return e
		}
		srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}
	ln, e := listen("tcp", srv.Addr)
	if e != nil {
		return e
	}
	errc := make(chan error, 1)
	go func() {
		if srv.TLSConfig != nil {
			errc <- srv.ServeTLS(ln, "", "")
			return
		}
		errc <- srv.Serve(ln)
	}()
	select {
	case e := <-errc:
		// serving failed before any shutdown was asked for
		return e
	case <-ctx.Done():
	}
	atomic.StoreInt32(&sd.draining, 1)
	rtr.log.Out("SERVER_SHUTDOWN", "Draining")
	time.Sleep(sd.delay)
	sctx, cancel := context.WithTimeout(context.Background(), sd.timeout)
	defer cancel()
	e = srv.Shutdown(sctx)
	// http.ErrServerClosed once Shutdown is called; anything else is a real serve error
	if serr := <-errc; e == nil && serr != nil && !errors.Is(serr, http.ErrServerClosed) {
		e = serr
	}
	return e
}

// shutdownSettings - the router's shutdown settings, created on first use
func (rtr *Router) shutdownSettings() *shutdown {
	if rtr.shutdown == nil {
		rtr.shutdown = &shutdown{timeout: DefaultShutdownTimeout}
	}
	return rtr.shutdown
}
