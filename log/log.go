// Package log implements simple key-value logging package.
//
// Package defines two log levels:
// - debug messages used for debug purposes
// - error messages that contains important information that require attention
//
// Output is formatted as flat JSON object containing only string values.
package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"time"
)

var root = newLogger(os.Stdout, 3)

// Error logs a message at level Error on the standard logger.
func Error(msg string, keyvals ...string) {
	root.Error(msg, keyvals...)
}

// Debug logs a message at level Debug on the standard logger.
func Debug(msg string, keyvals ...string) {
	root.Debug(msg, keyvals...)
}

// Fatal is equivalent to Error() followed by a call to os.Exit(1).
func Fatal(msg string, keyvals ...string) {
	root.Error(msg, keyvals...)
	os.Exit(1)
}

type logger struct {
	callDepth int
	write     func(interface{}) error
}

func newLogger(w io.Writer, callDepth int) *logger {
	return &logger{
		callDepth: callDepth,
		write:     json.NewEncoder(w).Encode,
	}
}

func (l *logger) Error(msg string, keyvals ...string) {
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "")
	}
	keyvals = append(keyvals, "msg", msg, "level", "ERROR")
	l.log(keyvals)
}

func (l *logger) Debug(msg string, keyvals ...string) {
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "")
	}
	keyvals = append(keyvals, "msg", msg, "level", "DEBUG")
	l.log(keyvals)
}

func (l *logger) log(keyvals []string) {
	now := currentTime().UTC().Format(time.RFC3339)

	file := "???:0"
	if _, name, line, ok := runtime.Caller(l.callDepth); ok {
		_, name = path.Split(name)
		file = fmt.Sprintf("%s:%d", name, line)
	}

	keyvals = append(keyvals, "date", now, "file", file)
	pairs := convertToPairs(keyvals)
	if err := l.write(pairs); err != nil {
		fmt.Fprintf(os.Stderr, "cannot write log message: %s\n", err)
		fmt.Fprintf(os.Stderr, "%v\n", keyvals)
	}
}

// we want to mock current time in tests
var currentTime = time.Now

func convertToPairs(keyvals []string) map[string]string {
	pairs := make(map[string]string)
	for i := 0; i < len(keyvals); i += 2 {
		pairs[keyvals[i]] = keyvals[i+1]
	}

	return pairs
}
