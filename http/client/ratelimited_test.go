package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRateLimited(t *testing.T) {
	tests := []struct {
		name      string
		rateLimit float64
		burst     int
		wantErr   bool
	}{
		{
			name:      "valid rate limited client",
			rateLimit: 10.0,
			burst:     5,
			wantErr:   false,
		},
		{
			name:      "zero rate limit",
			rateLimit: 0.0,
			burst:     1,
			wantErr:   false,
		},
		{
			name:      "high rate limit",
			rateLimit: 1000.0,
			burst:     100,
			wantErr:   false,
		},
		{
			name:      "burst of zero",
			rateLimit: 10.0,
			burst:     0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRateLimited(tt.rateLimit, tt.burst)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRateLimited() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if client.Client == nil {
					t.Error("NewRateLimited() client.Client is nil")
				}
				if client.limiter == nil {
					t.Error("NewRateLimited() client.limiter is nil")
				}
			}
		})
	}
}

func TestNewRateLimitedHTTP2(t *testing.T) {
	tests := []struct {
		name      string
		rateLimit float64
		burst     int
		wantErr   bool
	}{
		{
			name:      "valid rate limited HTTP2 client",
			rateLimit: 10.0,
			burst:     5,
			wantErr:   false,
		},
		{
			name:      "zero rate limit HTTP2",
			rateLimit: 0.0,
			burst:     1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRateLimitedHTTP2(tt.rateLimit, tt.burst)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRateLimitedHTTP2() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if client.Client == nil {
					t.Error("NewRateLimitedHTTP2() client.Client is nil")
				}
				if client.limiter == nil {
					t.Error("NewRateLimitedHTTP2() client.limiter is nil")
				}
			}
		})
	}
}

func TestRateLimitedClient_RateLimiting(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create a rate-limited client with 2 requests per second and burst of 1
	client, err := NewRateLimited(2.0, 1)
	if err != nil {
		t.Fatalf("Failed to create rate-limited client: %v", err)
	}
	client.SetJSON(true)

	ctx := context.Background()

	// Make first request (should be immediate due to burst)
	start := time.Now()
	res1 := client.Get(ctx, server.URL, nil, nil)
	elapsed1 := time.Since(start)

	if res1.Error != nil {
		t.Errorf("First request failed: %v", res1.Error)
	}

	// First request should be fast (within burst)
	if elapsed1 > 100*time.Millisecond {
		t.Errorf("First request took too long: %v (should be fast due to burst)", elapsed1)
	}

	// Make second request (should be rate-limited)
	start = time.Now()
	res2 := client.Get(ctx, server.URL, nil, nil)
	elapsed2 := time.Since(start)

	if res2.Error != nil {
		t.Errorf("Second request failed: %v", res2.Error)
	}

	// Second request should be delayed (rate limiting)
	// With rate limit of 2/sec, each request should be at least ~500ms apart
	// But burst of 1 means first request uses the burst, second should wait ~500ms
	if elapsed2 < 400*time.Millisecond {
		t.Errorf("Second request was not rate-limited: took %v (expected at least ~500ms)", elapsed2)
	}
}

func TestRateLimitedClient_ContextCancellation(t *testing.T) {
	// Create a test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create a rate-limited client with very low rate (1 request per 10 seconds)
	client, err := NewRateLimited(0.1, 0)
	if err != nil {
		t.Fatalf("Failed to create rate-limited client: %v", err)
	}

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// This request should fail due to context cancellation during rate limiting wait
	res := client.Get(ctx, server.URL, nil, nil)

	if res.Error == nil {
		t.Error("Expected error due to context cancellation, got nil")
	}
}

func TestRateLimitedClient_Methods(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"method": "` + r.Method + `"}`))
	}))
	defer server.Close()

	client, err := NewRateLimited(100.0, 10) // High rate limit for testing
	if err != nil {
		t.Fatalf("Failed to create rate-limited client: %v", err)
	}
	client.SetJSON(true)

	ctx := context.Background()
	testData := []byte(`{"test": "data"}`)

	tests := []struct {
		name   string
		method func(context.Context, string, []byte, map[string]string) Res
	}{
		{"Get", client.Get},
		{"Post", client.Post},
		{"Put", client.Put},
		{"Patch", client.Patch},
		{"Delete", client.Delete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.method(ctx, server.URL, testData, nil)
			if res.Error != nil {
				t.Errorf("%s() error = %v", tt.name, res.Error)
			}
			if res.Response.StatusCode != http.StatusOK {
				t.Errorf("%s() status code = %d, want %d", tt.name, res.Response.StatusCode, http.StatusOK)
			}
		})
	}
}

func TestRateLimitedClient_SetTimeout(t *testing.T) {
	client, err := NewRateLimited(10.0, 5)
	if err != nil {
		t.Fatalf("Failed to create rate-limited client: %v", err)
	}

	client.SetTimeout(5000)

	// Verify timeout was set on underlying client
	if client.Client.timeout != 5000 {
		t.Errorf("SetTimeout() timeout = %d, want %d", client.Client.timeout, 5000)
	}
}

func TestRateLimitedClient_SetLogger(t *testing.T) {
	client, err := NewRateLimited(10.0, 5)
	if err != nil {
		t.Fatalf("Failed to create rate-limited client: %v", err)
	}

	// Set logger to "empty" mode
	client.SetLogger("empty")

	// Verify SetLogger doesn't panic and can be called multiple times
	client.SetLogger("standard")
	client.SetLogger("erroronly")
}

func TestRateLimitedClient_SetJSON(t *testing.T) {
	client, err := NewRateLimited(10.0, 5)
	if err != nil {
		t.Fatalf("Failed to create rate-limited client: %v", err)
	}

	client.SetJSON(false)

	// Verify JSON setting was applied
	if client.Client.JSON != false {
		t.Errorf("SetJSON() JSON = %v, want %v", client.Client.JSON, false)
	}

	client.SetJSON(true)
	if client.Client.JSON != true {
		t.Errorf("SetJSON() JSON = %v, want %v", client.Client.JSON, true)
	}
}

func TestRateLimitedClient_BurstBehavior(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create a rate-limited client with burst of 3
	client, err := NewRateLimited(10.0, 3)
	if err != nil {
		t.Fatalf("Failed to create rate-limited client: %v", err)
	}

	ctx := context.Background()

	// Make 3 requests quickly - should all be fast due to burst
	start := time.Now()
	for i := 0; i < 3; i++ {
		res := client.Get(ctx, server.URL, nil, nil)
		if res.Error != nil {
			t.Errorf("Request %d failed: %v", i+1, res.Error)
		}
	}
	elapsed := time.Since(start)

	// All 3 requests should be fast (within burst)
	if elapsed > 200*time.Millisecond {
		t.Errorf("Burst requests took too long: %v (should be fast)", elapsed)
	}

	// 4th request should be rate-limited
	start = time.Now()
	res4 := client.Get(ctx, server.URL, nil, nil)
	elapsed4 := time.Since(start)

	if res4.Error != nil {
		t.Errorf("4th request failed: %v", res4.Error)
	}

	// 4th request should be delayed
	if elapsed4 < 50*time.Millisecond {
		t.Errorf("4th request was not rate-limited: took %v (expected delay)", elapsed4)
	}
}
