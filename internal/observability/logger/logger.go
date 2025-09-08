package logger

import "time"

// Logger defines minimal logging contract used across the service
type Logger interface {
	Info(event string, data map[string]interface{})
	Error(event string, data map[string]interface{})
	ExternalAPI(apiName, endpoint string, statusCode int, duration time.Duration, metadata map[string]interface{}) error
	Close() error
}

// NoopLogger is a safe default logger implementation
type NoopLogger struct{}

func (NoopLogger) Info(string, map[string]interface{})  {}
func (NoopLogger) Error(string, map[string]interface{}) {}
func (NoopLogger) ExternalAPI(string, string, int, time.Duration, map[string]interface{}) error {
	return nil
}
func (NoopLogger) Close() error { return nil }
