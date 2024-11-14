package log

import (
	"fmt"
)

// ExtendedLog is a struct that extends the Log struct
type ExtendedLog struct {
	Log
}

// ContextData is a struct that holds the logging context data
type ContextData struct {
	TraceID string
	Tenant  string
	UserID  string
}

// NewExtended creates a new ExtendedLog struct
func NewExtended(logtype string) (ExtendedLog, error) {
	var l ExtendedLog
	var e error
	if !isValid(logtype) {
		e = fmt.Errorf("invalid extended log type: %s", logtype)
		return l, e
	}
	l.config = logtype
	l.json = true
	return l, e
}

// Error logs an error message
func (l ExtendedLog) Error(scope string, ctx ContextData, errorMsg error, data interface{}, message string) {
	l.Log.Error(fmt.Sprintf("error:%s", scope), GetErrorLogMessage(ctx.TraceID, ctx.Tenant, ctx.UserID, errorMsg, data, message))
}

// Info logs an info message
func (l ExtendedLog) Info(scope string, ctx ContextData, data interface{}, message string) {
	l.Log.Out(fmt.Sprintf("info:%s", scope), GetLogMessage(ctx.TraceID, ctx.Tenant, ctx.UserID, data, message))
}

// Success logs a success message
func (l ExtendedLog) Success(scope string, ctx ContextData, data interface{}, message string) {
	l.Log.Out(fmt.Sprintf("ok:%s", scope), GetLogMessage(ctx.TraceID, ctx.Tenant, ctx.UserID, data, message))
}

// Debug logs a debug message
func (l ExtendedLog) Debug(scope string, ctx ContextData, data interface{}, message string) {
	l.Log.Debug(fmt.Sprintf("debug:%s", scope), GetLogMessage(ctx.TraceID, ctx.Tenant, ctx.UserID, data, message))
}
