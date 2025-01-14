package client

import (
	"errors"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/kelchy/go-lib/log"
)

func TestHTMLparse(t *testing.T) {
	tests := []struct {
		name       string
		response   *http.Response
		initialErr error
		expectHTML string
		expectErr  bool
	}{
		{
			name: "valid HTML response",
			response: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString("<html><body>Hello, World!</body></html>")),
			},
			expectHTML: "<html><body>Hello, World!</body></html>",
			expectErr:  false,
		},
		{
			name: "empty HTML response",
			response: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString("")),
			},
			expectHTML: "",
			expectErr:  false,
		},
		{
			name: "response with non-HTML content",
			response: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString("This is plain text")),
			},
			expectHTML: "This is plain text",
			expectErr:  false,
		},
		{
            name: "response with error in copying body",
            response: &http.Response{
                Body: ioutil.NopCloser(&errorReader{}),
            },
            initialErr: nil,
            expectHTML: "",
            expectErr:  true,
        },
        {
            name: "initial error in Res struct",
            response: &http.Response{
                Body: ioutil.NopCloser(bytes.NewBufferString("<html><body>Hello, World!</body></html>")),
            },
            initialErr: errors.New("initial error"),
            expectHTML: "",
            expectErr:  true,
        },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &Res{
				Response: *tt.response,
				Error:    tt.initialErr,
				log:      log.Log{},
			}
			res.HTMLparse()
			if res.HTML != tt.expectHTML {
				t.Errorf("Expected HTML %s, got %s", tt.expectHTML, res.HTML)
			}
			if (res.Error != nil) != tt.expectErr {
				t.Errorf("Expected error %v, got %v", tt.expectErr, res.Error != nil)
			}
		})
	}
}

// errorReader is a helper type to simulate an error when reading the response body
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
    return 0, errors.New("read error")
}

func (e *errorReader) Close() error {
    return nil
}

func TestJSONparse(t *testing.T) {
	tests := []struct {
		name       string
		response   *http.Response
		initialErr error
		expectJSON json.RawMessage
		expectErr  bool
	}{
		{
            name: "valid JSON response",
            response: &http.Response{
                Body: ioutil.NopCloser(bytes.NewBufferString(`{"message": "Hello, World!"}`)),
            },
            initialErr: nil,
            expectJSON: json.RawMessage(`{"message": "Hello, World!"}`),
            expectErr:  false,
        },
        {
            name: "invalid JSON response",
            response: &http.Response{
                Body: ioutil.NopCloser(bytes.NewBufferString(`{message: "Hello, World!"}`)),
            },
            initialErr: nil,
            expectJSON: nil,
            expectErr:  true,
        },
        {
            name: "empty JSON response",
            response: &http.Response{
                Body: ioutil.NopCloser(bytes.NewBufferString("")),
            },
            initialErr: nil,
            expectJSON: nil,
            expectErr:  true,
        },
        {
            name: "response with error in decoding body",
            response: &http.Response{
                Body: ioutil.NopCloser(&errorReader{}),
            },
            initialErr: nil,
            expectJSON: nil,
            expectErr:  true,
        },
        {
            name: "initial error in Res struct",
            response: &http.Response{
                Body: ioutil.NopCloser(bytes.NewBufferString(`{"message": "Hello, World!"}`)),
            },
            initialErr: errors.New("initial error"),
            expectJSON: nil,
            expectErr:  true,
        },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &Res{
				Response: *tt.response,
				Error:    tt.initialErr,
				log:      log.Log{},
			}
			res.JSONparse()
			if !bytes.Equal(res.JSON, tt.expectJSON) {
				t.Errorf("Expected JSON %s, got %s", tt.expectJSON, res.JSON)
			}
			if (res.Error != nil) != tt.expectErr {
				t.Errorf("Expected error %v, got %v", tt.expectErr, res.Error != nil)
			}
		})
	}
}
