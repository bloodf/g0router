package logging

import (
	"fmt"

	"github.com/bloodf/g0router/internal/store"
)

type RequestStore interface {
	LogRequest(entry *store.RequestLogEntry) error
}

type Logger struct {
	store RequestStore
}

func NewLogger(store RequestStore) *Logger {
	return &Logger{store: store}
}

func (l *Logger) Log(log RequestLog) error {
	entry := log.Entry()
	if err := l.store.LogRequest(&entry); err != nil {
		return fmt.Errorf("log request: %w", err)
	}

	return nil
}
