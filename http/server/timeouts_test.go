package server

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestApplyTimeoutsDefaults(t *testing.T) {
	rtr, _ := New(nil, nil)
	srv := &http.Server{}
	rtr.applyTimeouts(srv)
	if srv.ReadHeaderTimeout != 5*time.Second || srv.ReadTimeout != 10*time.Second ||
		srv.WriteTimeout != 10*time.Second || srv.IdleTimeout != 60*time.Second {
		t.Fatalf("defaults not applied: %+v", srv)
	}
}

func TestSetTimeoutsOverrides(t *testing.T) {
	rtr, _ := New(nil, nil)
	rtr.SetTimeouts(Timeouts{ReadHeader: time.Second, Read: 2 * time.Second, Write: 0, Idle: 3 * time.Second})
	srv := &http.Server{}
	rtr.applyTimeouts(srv)
	if srv.ReadHeaderTimeout != time.Second || srv.ReadTimeout != 2*time.Second ||
		srv.WriteTimeout != 0 || srv.IdleTimeout != 3*time.Second {
		t.Fatalf("overrides not applied: %+v", srv)
	}
	// a copy of the router made after SetTimeouts keeps them, as Run uses a copy
	cp := *rtr
	srv2 := &http.Server{}
	cp.applyTimeouts(srv2)
	if srv2.ReadHeaderTimeout != time.Second {
		t.Fatalf("copy lost the timeouts: %+v", srv2)
	}
}

func TestReadHeaderTimeoutClosesSlowClient(t *testing.T) {
	rtr, _ := New(nil, nil)
	rtr.SetTimeouts(Timeouts{ReadHeader: 100 * time.Millisecond})
	rtr.Get("/ok", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := run(rtr, ctx, "http", addr)
	defer func() {
		cancel()
		if e := <-done; e != nil {
			t.Error(e)
		}
	}()
	waitUp(t, &http.Client{Timeout: time.Second}, "http://"+addr+"/ok")

	conn, e := net.Dial("tcp", addr)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = conn.Close() }()
	// start a request but never finish its headers
	if _, e := conn.Write([]byte("GET /ok HTTP/1.1\r\nHost: x\r\n")); e != nil {
		t.Fatal(e)
	}
	if e := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); e != nil {
		t.Fatal(e)
	}
	start := time.Now()
	_, e = bufio.NewReader(conn).ReadByte()
	if e == nil {
		// the server may answer 408 before closing; either way it gave up on us
		if time.Since(start) > time.Second {
			t.Fatal("server answered too late")
		}
		return
	}
	if ne, ok := e.(net.Error); ok && ne.Timeout() {
		t.Fatal("server kept the slow connection open past ReadHeader")
	}
}
