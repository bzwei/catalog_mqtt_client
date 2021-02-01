package logger

import (
	"context"
	"fmt"
	"runtime"

	log "github.com/sirupsen/logrus"
)

type Logger int

const loggerID = "logger_id"

func (l Logger) Printf(s string, args ...interface{}) {
	log.Printf("[%s] [id=%d] %s", callerInfo(), l, fmt.Sprintf(s, args...))
}

func (l Logger) Println(s string) {
	log.Printf("[%s] [id=%d] %s", callerInfo(), l, s)
}

func (l Logger) Infof(s string, args ...interface{}) {
	x := fmt.Sprintf(s, args...)
	log.Infof("[%s] [id=%d] %s", callerInfo(), l, x)
}

func (l Logger) Info(s string) {
	log.Infof("[%s] [id=%d] %s", callerInfo(), l, s)
}

func (l Logger) Errorf(s string, args ...interface{}) {
	log.Errorf("[%s] [id=%d] %s", callerInfo(), l, fmt.Sprintf(s, args...))
}

func (l Logger) Error(s string) {
	log.Errorf("[%s] [id=%d] %s", callerInfo(), l, s)
}

func CtxWithLoggerID(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, loggerID, id)
}

func GetLogger(ctx context.Context) Logger {
	return Logger(ctx.Value(loggerID).(int))
}

func callerInfo() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "missing"
	}
	return fmt.Sprintf("%s:%d", file, line)
}
