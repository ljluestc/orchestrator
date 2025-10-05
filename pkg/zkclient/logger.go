package zkclient

import (
	"log"
	"os"
)

// Logger is an interface for logging
type Logger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

// defaultLogger is a simple logger implementation using standard log package
type defaultLogger struct {
	logger *log.Logger
}

func (l *defaultLogger) Debug(format string, v ...interface{}) {
	if l.logger == nil {
		l.logger = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags)
	}
	l.logger.Printf(format, v...)
}

func (l *defaultLogger) Info(format string, v ...interface{}) {
	if l.logger == nil {
		l.logger = log.New(os.Stdout, "[INFO] ", log.LstdFlags)
	}
	l.logger.Printf(format, v...)
}

func (l *defaultLogger) Warn(format string, v ...interface{}) {
	if l.logger == nil {
		l.logger = log.New(os.Stdout, "[WARN] ", log.LstdFlags)
	}
	l.logger.Printf(format, v...)
}

func (l *defaultLogger) Error(format string, v ...interface{}) {
	if l.logger == nil {
		l.logger = log.New(os.Stderr, "[ERROR] ", log.LstdFlags)
	}
	l.logger.Printf(format, v...)
}

// NoOpLogger is a logger that does nothing
type NoOpLogger struct{}

func (l *NoOpLogger) Debug(format string, v ...interface{}) {}
func (l *NoOpLogger) Info(format string, v ...interface{})  {}
func (l *NoOpLogger) Warn(format string, v ...interface{})  {}
func (l *NoOpLogger) Error(format string, v ...interface{}) {}
