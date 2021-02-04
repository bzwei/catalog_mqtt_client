package logger

import (
	"context"
	"fmt"
	"runtime"

	log "github.com/sirupsen/logrus"
)

// Logger provides a set of log methods that embeds a log ID and caller info in the message
type Logger string

type key int

const loggerID key = 1

// Printf logs a message with variables at level Info
func (l Logger) Printf(s string, args ...interface{}) {
	log.Printf("[%s] [id=%s] %s", callerInfo(), l, fmt.Sprintf(s, args...))
}

// Println logs a message at level Info
func (l Logger) Println(s string) {
	log.Printf("[%s] [id=%s] %s", callerInfo(), l, s)
}

// Infof logs a message with variables at level Info
func (l Logger) Infof(s string, args ...interface{}) {
	x := fmt.Sprintf(s, args...)
	log.Infof("[%s] [id=%s] %s", callerInfo(), l, x)
}

// Info logs a message at level Info
func (l Logger) Info(s string) {
	log.Infof("[%s] [id=%s] %s", callerInfo(), l, s)
}

// Errorf logs a message with variables at level Error
func (l Logger) Errorf(s string, args ...interface{}) {
	log.Errorf("[%s] [id=%s] %s", callerInfo(), l, fmt.Sprintf(s, args...))
}

// Error logs a message at level Error
func (l Logger) Error(s string) {
	log.Errorf("[%s] [id=%s] %s", callerInfo(), l, s)
}

// CtxWithLoggerID creates a new context from parent Context ctx and stores the id as loggerID in the context
func CtxWithLoggerID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, loggerID, id)
}

// GetLogger extracts the Logger from Context ctx
func GetLogger(ctx context.Context) Logger {
	return Logger(ctx.Value(loggerID).(string))
}

func callerInfo() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "missing"
	}
	return fmt.Sprintf("%s:%d", file, line)
}
