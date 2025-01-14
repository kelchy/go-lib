package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kelchy/go-lib/log"
)

func TestNew(t *testing.T) {
	// Test case with empty origins and headers
	router, err := New(nil, nil)
	if err != nil {
		t.Fatalf("Error creating router: %v", err)
	}
	if router == nil {
		t.Fatal("Router is nil")
	}

	// Test case with invalid log configuration
	_, err = log.New("invalid config")
	if err == nil {
		t.Fatal("Expected an error creating logger")
	}

	// happy flow
	origins := []string{"http://localhost"}
	headers := []string{"Accept", "Content-Type"}

	router, err = New(origins, headers)
	if err != nil {
		t.Fatalf("Error creating router: %v", err)
	}

	if router == nil {
		t.Fatal("Router is nil")
	}

	// Test that the router has the expected fields
	if router.log == (log.Log{}) {
		t.Fatal("Log is nil/invalid")
	}
	if router.logRequest != true {
		t.Fatal("LogRequest should be true by default")
	}
	if len(router.logSkipPath) != 1 || router.logSkipPath[0] != "/" {
		t.Fatal("LogSkipPath should have the default value")
	}
	if router.Engine == nil {
		t.Fatal("Engine is nil")
	}

	// Test that the router middleware has been set correctly
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp := httptest.NewRecorder()
	router.Engine.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatal("catchall middleware should return 404")
	}

	// Test case with empty origins and default headers
	router, err = New(nil, []string{})
	if err != nil {
		t.Fatalf("Error creating router: %v", err)
	}
	if router == nil {
		t.Fatal("Router is nil")
	}

	// Test case with default origins and empty headers
	router, err = New([]string{}, nil)
	if err != nil {
		t.Fatalf("Error creating router: %v", err)
	}
	if router == nil {
		t.Fatal("Router is nil")
	}
}

func TestNewWithCorsOptions(t *testing.T) {
	// Test case with empty origins and headers and nil AllowedOriginFunc
	corsOptions := &CorsOptions{
		Origins:           nil,
		Headers:           nil,
		AllowedOriginFunc: nil,
	}
	router, err := NewWithCorsOptions(corsOptions)
	if err != nil {
		t.Fatalf("Error creating router: %v", err)
	}
	if router == nil {
		t.Fatal("Router is nil")
	}
	// Test case with empty origins and headers and nil AllowedOriginFunc
	corsOptions = &CorsOptions{
		Origins:           []string{},
		Headers:           []string{},
		AllowedOriginFunc: nil,
	}
	router, err = NewWithCorsOptions(corsOptions)
	if err != nil {
		t.Fatalf("Error creating router: %v", err)
	}
	if router == nil {
		t.Fatal("Router is nil")
	}
	// Test case with empty origins and headers and AllowedOriginFunc
	corsOptions = &CorsOptions{
		Origins:           []string{},
		Headers:           []string{},
		AllowedOriginFunc: func(r *http.Request, origin string) bool { return true },
	}
	router, err = NewWithCorsOptions(corsOptions)
	if err != nil {
		t.Fatalf("Error creating router: %v", err)
	}
	if router == nil {
		t.Fatal("Router is nil")
	}
	// Test case with all values present
	corsOptions = &CorsOptions{
		Origins:           []string{"http://localhost"},
		Headers:           []string{"Accept", "Content-Type"},
		AllowedOriginFunc: func(r *http.Request, origin string) bool { return true },
	}
	router, err = NewWithCorsOptions(corsOptions)
	if err != nil {
		t.Fatalf("Error creating router: %v", err)
	}
	if router == nil {
		t.Fatal("Router is nil")
	}
}
