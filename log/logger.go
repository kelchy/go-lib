package log

import (
	"fmt"
)

type ExtendedLog struct {
	Log
}

type ContextData struct {
	TraceID string
	Tenant string
	UserID string
}


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

func (l ExtendedLog) ErrorLog(scope string, ctx ContextData, errorMsg error, data interface{}, message string) {
	l.Error(fmt.Sprintf("error:%s", scope), GetErrorLogMessage(ctx.TraceID, ctx.Tenant, ctx.UserID, errorMsg, data, message))
}

func (l ExtendedLog) InfoLog(scope string, ctx ContextData, data interface{}, message string) {
	l.Out(fmt.Sprintf("info:%s", scope), GetLogMessage(ctx.TraceID, ctx.Tenant, ctx.UserID, data, message))
}

func (l ExtendedLog) SuccessLog(scope string, ctx ContextData, data interface{}, message string) {
	l.Out(fmt.Sprintf("ok:%s", scope), GetLogMessage(ctx.TraceID, ctx.Tenant, ctx.UserID, data, message))
}

func (l ExtendedLog) DebugLog(scope string, ctx ContextData, data interface{}, message string) {
	l.Debug(fmt.Sprintf("debug:%s", scope), GetLogMessage(ctx.TraceID, ctx.Tenant, ctx.UserID, data, message))
}