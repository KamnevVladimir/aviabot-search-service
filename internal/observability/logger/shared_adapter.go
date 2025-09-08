package logger

import (
    shared "github.com/KamnevVladimir/aviabot-shared-logging"
    "time"
)

// SharedAdapter wraps shared logging client to our local interface
type SharedAdapter struct {
    c *shared.Client
}

func NewSharedAdapter(c *shared.Client) *SharedAdapter { return &SharedAdapter{c: c} }

func (s *SharedAdapter) Info(event string, data map[string]interface{})  { _ = s.c.Info(event, "", data) }
func (s *SharedAdapter) Error(event string, data map[string]interface{}) { _ = s.c.Error(nil, event, data) }

func (s *SharedAdapter) ExternalAPI(apiName, endpoint string, statusCode int, duration time.Duration, metadata map[string]interface{}) error {
    return s.c.ExternalAPI(apiName, endpoint, statusCode, duration, metadata)
}

func (s *SharedAdapter) Close() error { return nil }


