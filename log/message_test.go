package log

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGetLogMessage(t *testing.T) {
	tests := []struct {
		traceID     string
		tenant      string
		externalID  string
		data        interface{}
		message     string
		expectedMsg string
	}{
		{
			traceID:     "trace123",
			tenant:      "tenantABC",
			externalID:  "external456",
			data:        "some data",
			message:     "a message",
			expectedMsg: "traceID:trace123; tenant:tenantABC; userID:external456; data:some data; message:a message",
		},
		{
			traceID:     "",
			tenant:      "",
			externalID:  "",
			data:        nil,
			message:     "",
			expectedMsg: "",
		},
		{
			traceID:     "trace456",
			tenant:      "tenantXYZ",
			externalID:  "external789",
			data:        map[string]string{"key": "value"},
			message:     "another message",
			expectedMsg: "traceID:trace456; tenant:tenantXYZ; userID:external789; data:map[key:value]; message:another message",
		},
		{
			traceID:     "trace789",
			tenant:      "tenantDEF",
			externalID:  "external012",
			data:        12345,
			message:     "integer data",
			expectedMsg: "traceID:trace789; tenant:tenantDEF; userID:external012; data:12345; message:integer data",
		},
		{
			traceID:     "trace123456\nnewline",
			tenant:      "tenantABC\ttab",
			externalID:  "external456\bspecial",
			data:        "some\ndata",
			message:     "a message\twith special chars",
			expectedMsg: "traceID:trace123456\nnewline; tenant:tenantABC\ttab; userID:external456\bspecial; data:some\ndata; message:a message\twith special chars",
		},
		{
			traceID:     "trace" + strings.Repeat("1234567", 1000),
			tenant:      "tenant" + strings.Repeat("ABC", 1000),
			externalID:  "external" + strings.Repeat("456", 1000),
			data:        "some data" + strings.Repeat("...", 1000),
			message:     "a message" + strings.Repeat(".", 1000),
			expectedMsg: fmt.Sprintf("traceID:trace%s; tenant:tenant%s; userID:external%s; data:some data%s; message:a message%s", strings.Repeat("1234567", 1000), strings.Repeat("ABC", 1000), strings.Repeat("456", 1000), strings.Repeat("...", 1000), strings.Repeat(".", 1000)),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("traceID:%s, tenant:%s", tt.traceID, tt.tenant), func(t *testing.T) {
			log := GetLogMessage(tt.traceID, tt.tenant, tt.externalID, tt.data, tt.message)
			if log != tt.expectedMsg {
				t.Errorf("expected %s but got %s", tt.expectedMsg, log)
			}
		})
	}
}

func TestGetErrorLogMessage(t *testing.T) {
	tests := []struct {
		traceID     string
		tenant      string
		externalID  string
		errorMsg    error
		data        interface{}
		message     string
		expectedMsg string
	}{
		{
			traceID:     "trace123",
			tenant:      "tenantABC",
			externalID:  "external456",
			errorMsg:    errors.New("an error occurred"),
			data:        "some data",
			message:     "a message",
			expectedMsg: "traceID:trace123; tenant:tenantABC; userID:external456; data:some data; message:a message; error:an error occurred",
		},
		{
			traceID:     "",
			tenant:      "",
			externalID:  "",
			errorMsg:    errors.New("empty error"),
			data:        nil,
			message:     "",
			expectedMsg: "error:empty error",
		},
		{
			traceID:     "trace1234",
			tenant:      "tenantABC",
			externalID:  "external456",
			errorMsg:    nil,
			data:        "some data",
			message:     "a message",
			expectedMsg: "traceID:trace1234; tenant:tenantABC; userID:external456; data:some data; message:a message; error:<nil>",
		},
		{
			traceID:     "trace12345",
			tenant:      "tenantABC",
			externalID:  "external456",
			errorMsg:    errors.New("an error occurred"),
			data:        nil,
			message:     "a message",
			expectedMsg: "traceID:trace12345; tenant:tenantABC; userID:external456; message:a message; error:an error occurred",
		},
		{
			traceID:     "trace123456",
			tenant:      "tenantABC",
			externalID:  "external456",
			errorMsg:    errors.New("an error occurred"),
			data:        map[string]string{"key": "value"},
			message:     "a message",
			expectedMsg: "traceID:trace123456; tenant:tenantABC; userID:external456; data:map[key:value]; message:a message; error:an error occurred",
		},
		{
			traceID:     "trace123456\nnewline",
			tenant:      "tenantABC\ttab",
			externalID:  "external456\bspecial",
			errorMsg:    errors.New("error\nnewline\terror"),
			data:        "some\ndata",
			message:     "a message\twith special chars",
			expectedMsg: "traceID:trace123456\nnewline; tenant:tenantABC\ttab; userID:external456\bspecial; data:some\ndata; message:a message\twith special chars; error:error\nnewline\terror",
		},
		{
			traceID:     "trace" + strings.Repeat("1234567", 1000),
			tenant:      "tenant" + strings.Repeat("ABC", 1000),
			externalID:  "external" + strings.Repeat("456", 1000),
			errorMsg:    errors.New("an error occurred" + strings.Repeat("!", 1000)),
			data:        "some data" + strings.Repeat("...", 1000),
			message:     "a message" + strings.Repeat(".", 1000),
			expectedMsg: fmt.Sprintf("traceID:trace%s; tenant:tenant%s; userID:external%s; data:some data%s; message:a message%s; error:an error occurred%s", strings.Repeat("1234567", 1000), strings.Repeat("ABC", 1000), strings.Repeat("456", 1000), strings.Repeat("...", 1000), strings.Repeat(".", 1000), strings.Repeat("!", 1000)),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("traceID:%s, tenant:%s", tt.traceID, tt.tenant), func(t *testing.T) {
			err := GetErrorLogMessage(tt.traceID, tt.tenant, tt.externalID, tt.errorMsg, tt.data, tt.message)
			if err == nil {
				t.Errorf("expected error but got nil")
			}
			if err != nil {
				if err.Error() != tt.expectedMsg {
					t.Errorf("expected %s but got %s", tt.expectedMsg, err.Error())
				}
			}
		})
	}
}
