package log

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestErrorLog(t *testing.T) {
	tests := []struct {
		scope       string
		traceID     string
		contextData ContextData
		errorMsg    error
		data        interface{}
		message     string
		expectedMsg string
	}{
		{
			scope:       "scope1",
			traceID:     "trace123",
			contextData: ContextData{TraceID: "trace123", UserID: "external456", Tenant: "tenantABC"},
			errorMsg:    errors.New("an error occurred"),
			data:        "some data",
			message:     "a message",
			expectedMsg: "\"scope\":\"error:scope1\",\"msg\":\"traceID:trace123; tenant:tenantABC; userID:external456; data:some data; message:a message; error:an error occurred",
		},
		{
			scope:       "scope2",
			contextData: ContextData{TraceID: "", UserID: "", Tenant: ""},
			errorMsg:    errors.New("another error"),
			data:        nil,
			message:     "",
			expectedMsg: "\"scope\":\"error:scope2\",\"msg\":\"error:another error",
		},
		{
			scope:       "scope3",
			contextData: ContextData{TraceID: "trace456", UserID: "external789", Tenant: "tenantXYZ"},
			errorMsg:    nil,
			data:        map[string]string{"key": "value"},
			message:     "another message",
			expectedMsg: "\"scope\":\"error:scope3\",\"msg\":\"traceID:trace456; tenant:tenantXYZ; userID:external789; data:map[key:value]; message:another message; error:\\u003cnil\\u003e",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("scope:%s", tt.scope), func(t *testing.T) {

			// Custom logger to capture log output
			var buf bytes.Buffer

			oldStderr := os.Stderr

			r, w, _ := os.Pipe()
			os.Stderr = w

			// Use a goroutine to copy the output from the pipe to the buffer
			done := make(chan struct{})
			go func() {
				_, _ = buf.ReadFrom(r)
				close(done)
			}()

			log, _ := NewExtended("standard")
			log.Error(tt.scope, tt.contextData, tt.errorMsg, tt.data, tt.message)

			_ = w.Close()
			<-done
			os.Stderr = oldStderr

			got := buf.String()
			if !strings.Contains(got, tt.expectedMsg) {
				t.Errorf("expected log message to contain %q, got %q", tt.expectedMsg, got)
			}
		})
	}

}

func TestInfoLog(t *testing.T) {
	tests := []struct {
		scope       string
		traceID     string
		contextData ContextData
		errorMsg    error
		data        interface{}
		message     string
		expectedMsg string
	}{
		{
			scope:       "scope1",
			traceID:     "trace123",
			contextData: ContextData{TraceID: "trace123", UserID: "external456", Tenant: "tenantABC"},
			data:        "some data",
			message:     "a message",
			expectedMsg: "\"scope\":\"info:scope1\",\"msg\":\"traceID:trace123; tenant:tenantABC; userID:external456; data:some data; message:a message",
		},
		{
			scope:       "scope2",
			contextData: ContextData{TraceID: "", UserID: "", Tenant: ""},
			data:        nil,
			message:     "",
			expectedMsg: "\"scope\":\"info:scope2\",\"msg\":\"\"",
		},
		{
			scope:       "scope3",
			contextData: ContextData{TraceID: "trace456", UserID: "external789", Tenant: "tenantXYZ"},
			data:        map[string]string{"key": "value"},
			message:     "another message",
			expectedMsg: "\"scope\":\"info:scope3\",\"msg\":\"traceID:trace456; tenant:tenantXYZ; userID:external789; data:map[key:value]; message:another message",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("scope:%s", tt.scope), func(t *testing.T) {

			// Custom logger to capture log output
			var buf bytes.Buffer

			oldStdout := os.Stdout

			r, w, _ := os.Pipe()
			os.Stdout = w

			// Use a goroutine to copy the output from the pipe to the buffer
			done := make(chan struct{})
			go func() {
				_, _ = buf.ReadFrom(r)
				close(done)
			}()

			log, _ := NewExtended("standard")
			log.Info(tt.scope, tt.contextData, tt.data, tt.message)

			_ = w.Close()
			os.Stdout = oldStdout
			<-done

			got := buf.String()
			if !strings.Contains(got, tt.expectedMsg) {
				t.Errorf("expected log message to contain %q, got %q", tt.expectedMsg, got)
			}
		})
	}

}

func TestOutLog(t *testing.T) {
	tests := []struct {
		scope       string
		traceID     string
		contextData ContextData
		errorMsg    error
		data        interface{}
		message     string
		expectedMsg string
	}{
		{
			scope:       "scope1",
			contextData: ContextData{TraceID: "trace123", UserID: "external456", Tenant: "tenantABC"},
			data:        "some data",
			message:     "a message",
			expectedMsg: "\"scope\":\"ok:scope1\",\"msg\":\"traceID:trace123; tenant:tenantABC; userID:external456; data:some data; message:a message",
		},
		{
			scope:       "scope2",
			traceID:     "",
			contextData: ContextData{TraceID: "", UserID: "", Tenant: ""},
			data:        nil,
			message:     "",
			expectedMsg: "\"scope\":\"ok:scope2\",\"msg\":\"\"",
		},
		{
			scope:       "scope3",
			traceID:     "trace456",
			contextData: ContextData{TraceID: "trace456", UserID: "external789", Tenant: "tenantXYZ"},
			data:        map[string]string{"key": "value"},
			message:     "another message",
			expectedMsg: "\"scope\":\"ok:scope3\",\"msg\":\"traceID:trace456; tenant:tenantXYZ; userID:external789; data:map[key:value]; message:another message",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("scope:%s", tt.scope), func(t *testing.T) {

			// Custom logger to capture log output
			var buf bytes.Buffer

			oldStdout := os.Stdout

			r, w, _ := os.Pipe()
			os.Stdout = w

			// Use a goroutine to copy the output from the pipe to the buffer
			done := make(chan struct{})
			go func() {
				_, _ = buf.ReadFrom(r)
				close(done)
			}()

			log, _ := NewExtended("standard")
			log.Out(tt.scope, tt.contextData, tt.data, tt.message)

			_ = w.Close()
			os.Stdout = oldStdout
			<-done

			got := buf.String()
			if !strings.Contains(got, tt.expectedMsg) {
				t.Errorf("expected log message to contain %q, got %q", tt.expectedMsg, got)
			}
		})
	}

}

func TestDebugLog(t *testing.T) {
	tests := []struct {
		scope       string
		traceID     string
		contextData ContextData
		errorMsg    error
		data        interface{}
		message     string
		expectedMsg string
	}{
		{
			scope:       "scope1",
			contextData: ContextData{TraceID: "trace123", UserID: "external456", Tenant: "tenantABC"},
			data:        "some data",
			message:     "a message",
			expectedMsg: "\"scope\":\"debug:scope1\",\"msg\":\"traceID:trace123; tenant:tenantABC; userID:external456; data:some data; message:a message",
		},
		{
			scope:       "scope2",
			traceID:     "",
			contextData: ContextData{TraceID: "", UserID: "", Tenant: ""},
			data:        nil,
			message:     "",
			expectedMsg: "\"scope\":\"debug:scope2\",\"msg\":\"\"",
		},
		{
			scope:       "scope3",
			traceID:     "trace456",
			contextData: ContextData{TraceID: "trace456", UserID: "external789", Tenant: "tenantXYZ"},
			data:        map[string]string{"key": "value"},
			message:     "another message",
			expectedMsg: "\"scope\":\"debug:scope3\",\"msg\":\"traceID:trace456; tenant:tenantXYZ; userID:external789; data:map[key:value]; message:another message",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("scope:%s", tt.scope), func(t *testing.T) {

			// Custom logger to capture log output
			var buf bytes.Buffer

			oldStdout := os.Stdout

			r, w, _ := os.Pipe()
			os.Stdout = w

			// Use a goroutine to copy the output from the pipe to the buffer
			done := make(chan struct{})
			go func() {
				_, _ = buf.ReadFrom(r)
				close(done)
			}()

			log, _ := NewExtended("standard")
			log.Debug(tt.scope, tt.contextData, tt.data, tt.message)

			_ = w.Close()
			os.Stdout = oldStdout
			<-done

			got := buf.String()
			if !strings.Contains(got, tt.expectedMsg) {
				t.Errorf("expected log message to contain %q, got %q", tt.expectedMsg, got)
			}
		})
	}

}
