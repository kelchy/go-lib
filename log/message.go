package log

import (
	"fmt"
	"strings"
)

// GetLogMessage composes log message
func GetLogMessage(traceID string, tenant string, userID string, data interface{}, message string) string {
	var parts []string

	if traceID != "" {
		parts = append(parts, fmt.Sprintf("traceID:%s", traceID))
	}
	if tenant != "" {
		parts = append(parts, fmt.Sprintf("tenant:%s", tenant))
	}
	if userID != "" {
		parts = append(parts, fmt.Sprintf("userID:%s", userID))
	}
	if data != nil {
		parts = append(parts, fmt.Sprintf("data:%+v", data))
	}
	if message != "" {
		parts = append(parts, fmt.Sprintf("message:%s", message))
	}

	return strings.Join(parts, "; ")
}

// GetErrorLogMessage composes error log message
func GetErrorLogMessage(traceID string, tenant string, userID string, errorMsg error, data interface{}, message string) error {
	var parts []string

	if traceID != "" {
		parts = append(parts, fmt.Sprintf("traceID:%s", traceID))
	}
	if tenant != "" {
		parts = append(parts, fmt.Sprintf("tenant:%s", tenant))
	}
	if userID != "" {
		parts = append(parts, fmt.Sprintf("userID:%s", userID))
	}
	if data != nil {
		parts = append(parts, fmt.Sprintf("data:%+v", data))
	}
	if message != "" {
		parts = append(parts, fmt.Sprintf("message:%s", message))
	}

	parts = append(parts, fmt.Sprintf("error:%+v", errorMsg))

	return fmt.Errorf(strings.Join(parts, "; "))
}
