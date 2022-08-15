package log

import (
	"os"
	"time"
	"errors"
	"fmt"
	"runtime/debug"
	"encoding/json"
)

// Log - instance created when initializing logger
type Log struct {
	config	string
	json	bool
}

// Line - json struct to indicate a logger line
type Line struct {
	Ts	time.Time	`json:"ts"`
	Scope	string		`json:"scope"`
	Msg	string		`json:"msg"`
	Stack	string		`json:"stack,omitempty"`
}

// New - constructor to create an instance of the logger
// returns the instance and error
func New(logtype string) (Log, error) {
	var l Log
	var e error
	if !isValid(logtype) {
		e = errors.New("Invalid log type")
		return l, e
	}
	l.config = logtype
	l.json = true
	return l, e
}

// Out - outputs to stdout
func (l Log) Out(scope string, msg string) {
	if l.config != "empty" && l.config != "erroronly" {
		logPrint(scope, msg, l.json)
	}
}

// Debug - outputs debugging to stdout
func (l Log) Debug(scope string, msg string) {
	if l.config != "empty" {
		logPrint(scope, msg, l.json)
	}
}

// Error - outputs to stderr
func (l Log) Error(scope string, err error) {
	if err != nil && l.config != "empty" {
		ts := time.Now()
		if !l.json {
			fmt.Fprintln(os.Stderr, ts.Format(time.RFC3339), scope, err)
			fmt.Fprintln(os.Stderr, string(debug.Stack()));
		} else {
			jline, e := log2Json(ts, scope, err.Error(), string(debug.Stack()))
			if e != nil {
				fmt.Fprintln(os.Stderr, ts.Format(time.RFC3339), scope, err.Error())
			} else {
				fmt.Fprintln(os.Stderr, jline)
			}
		}
	}
}

// JSONDisable - used to internally turn off json logging
func (l *Log) JSONDisable() {
	l.json = false
}

// JSONEnable - used to internally turn on json logging
func (l *Log) JSONEnable() {
	l.json = true
}

func logPrint(scope string, msg string, j bool) {
	ts := time.Now()
	if !j {
		fmt.Println(ts.Format(time.RFC3339), scope, msg)
	} else {
		jline, e := log2Json(ts, scope, msg, "")
		if e != nil {
			fmt.Println(ts.Format(time.RFC3339), scope, msg)
		} else {
			fmt.Println(jline)
		}
	}
}

func isValid(logtype string) bool {
	switch logtype {
	case "", "standard", "empty", "erroronly":
		return true
	}
	return false
}

func log2Json(ts time.Time, scope string, msg string, stack string) (string, error) {
	line := &Line{
		Ts: ts,
		Scope: scope,
		Msg: msg,
	}
	if (stack != "") {
		line.Stack = stack
	}
	j, e := json.Marshal(line)
	if e != nil {
		return "", e
	}
	return string(j), nil
}
