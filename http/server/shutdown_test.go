package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// freeAddr - a local address nothing listens on
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

// waitUp - waits for the server at addr to answer
func waitUp(t *testing.T, client *http.Client, url string) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if res, e := client.Get(url); e == nil {
			res.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server never came up")
}

// run - starts RunContext in the background and returns its result channel
func run(rtr *Router, ctx context.Context, proto string, addr string) chan error {
	done := make(chan error, 1)
	go func() { done <- rtr.RunContext(ctx, proto, addr) }()
	return done
}

func TestRunContextDrainsInFlightRequests(t *testing.T) {
	rtr, _ := New(nil, nil)
	rtr.SetShutdown(5*time.Second, 50*time.Millisecond)
	started := make(chan struct{}, 1)
	rtr.Get("/slow", func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	rtr.Get("/up", func(w http.ResponseWriter, r *http.Request) {})
	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := run(rtr, ctx, "http", addr)
	waitUp(t, http.DefaultClient, "http://"+addr+"/up")
	if rtr.Draining() {
		t.Fatal("draining before shutdown")
	}

	code := make(chan int, 1)
	go func() {
		res, e := http.Get("http://" + addr + "/slow")
		if e != nil {
			code <- 0
			return
		}
		res.Body.Close()
		code <- res.StatusCode
	}()
	<-started
	cancel()
	if c := <-code; c != http.StatusOK {
		t.Fatalf("in-flight request got %d, want 200", c)
	}
	if e := <-done; e != nil {
		t.Fatalf("graceful shutdown returned %v", e)
	}
	if !rtr.Draining() {
		t.Fatal("not draining after shutdown")
	}
}

func TestRunContextShutdownTimeout(t *testing.T) {
	rtr, _ := New(nil, nil)
	rtr.SetShutdown(20*time.Millisecond, 0)
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	rtr.Get("/stuck", func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-release
	})
	rtr.Get("/up", func(w http.ResponseWriter, r *http.Request) {})
	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := run(rtr, ctx, "http", addr)
	waitUp(t, http.DefaultClient, "http://"+addr+"/up")
	go func() {
		if res, e := http.Get("http://" + addr + "/stuck"); e == nil {
			res.Body.Close()
		}
	}()
	<-started
	cancel()
	e := <-done
	close(release)
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatalf("got %v, want a shutdown timeout", e)
	}
}

func TestRunContextH2C(t *testing.T) {
	rtr, _ := New(nil, nil)
	rtr.Get("/up", func(w http.ResponseWriter, r *http.Request) {})
	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := run(rtr, ctx, "h2c", addr)
	waitUp(t, http.DefaultClient, "http://"+addr+"/up")
	cancel()
	if e := <-done; e != nil {
		t.Fatalf("got %v", e)
	}
}

func TestRunContextErrors(t *testing.T) {
	rtr, _ := New(nil, nil)
	if e := rtr.RunContext(context.Background(), "ftp", ":0"); e == nil {
		t.Fatal("unknown proto should fail")
	}

	// address already in use
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer ln.Close()
	if e := rtr.RunContext(context.Background(), "http", ln.Addr().String()); e == nil {
		t.Fatal("busy address should fail")
	}
}

// brokenListener - a listener whose Accept fails for good
type brokenListener struct{ net.Listener }

func (brokenListener) Accept() (net.Conn, error) { return nil, errors.New("accept failed") }

func TestRunContextServeFails(t *testing.T) {
	defer func() { listen = net.Listen }()
	listen = func(network, addr string) (net.Listener, error) {
		ln, e := net.Listen(network, addr)
		return brokenListener{ln}, e
	}
	rtr, _ := New(nil, nil)
	if e := rtr.RunContext(context.Background(), "http", "127.0.0.1:0"); e == nil || e.Error() != "accept failed" {
		t.Fatalf("got %v, want the serve error", e)
	}
}

// writeCert - writes a self-signed certificate and key for 127.0.0.1
func writeCert(t *testing.T) (string, string) {
	t.Helper()
	priv, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, e := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if e != nil {
		t.Fatal(e)
	}
	kder, e := x509.MarshalECPrivateKey(priv)
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	crt, key := filepath.Join(dir, "crt.pem"), filepath.Join(dir, "key.pem")
	os.WriteFile(crt, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600)
	os.WriteFile(key, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kder}), 0o600)
	return crt, key
}

func TestRunSContext(t *testing.T) {
	rtr, _ := New(nil, nil)
	rtr.Get("/up", func(w http.ResponseWriter, r *http.Request) {})
	crt, key := writeCert(t)
	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- rtr.RunSContext(ctx, "https", addr, crt, key) }()
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	waitUp(t, client, "https://"+addr+"/up")
	cancel()
	if e := <-done; e != nil {
		t.Fatalf("got %v", e)
	}

	if e := rtr.RunSContext(context.Background(), "https", addr, "missing.pem", "missing.pem"); e == nil {
		t.Fatal("missing certificate should fail")
	}
	if e := rtr.RunSContext(context.Background(), "h2", addr, crt, key); e != nil {
		t.Fatalf("h2 is not implemented and returns nil, got %v", e)
	}
	if e := rtr.RunSContext(context.Background(), "ftp", addr, crt, key); e == nil {
		t.Fatal("unknown proto should fail")
	}
}

func TestRunAndRunSStopOnSignal(t *testing.T) {
	orig := signalContext
	defer func() { signalContext = orig }()
	signalContext = func() (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // the signal has already arrived
		return ctx, cancel
	}
	rtr, _ := New(nil, nil)
	if e := rtr.Run("http", freeAddr(t)); e != nil {
		t.Fatalf("Run got %v", e)
	}
	crt, key := writeCert(t)
	if e := rtr.RunS("https", freeAddr(t), crt, key); e != nil {
		t.Fatalf("RunS got %v", e)
	}

	// the real signal context is live until a signal arrives
	ctx, stop := orig()
	if ctx.Err() != nil {
		t.Fatal("signal context done before any signal")
	}
	stop()
}

func TestZeroRouterDefaults(t *testing.T) {
	var rtr Router
	if rtr.Draining() {
		t.Fatal("zero router should not be draining")
	}
	rtr.SetShutdown(time.Second, 0)
	if rtr.shutdown.timeout != time.Second {
		t.Fatal("SetShutdown on a zero router should create the settings")
	}

	// a router built without New still shuts down with the default timeout
	r2, _ := New(nil, nil)
	r2.shutdown = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := r2.RunContext(ctx, "http", freeAddr(t)); e != nil {
		t.Fatalf("got %v", e)
	}
}
