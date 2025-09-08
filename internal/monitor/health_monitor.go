package monitor

import (
	"context"
	"time"
)

// Logger is a minimal logger used by health monitor
type Logger interface {
	Info(event string, data map[string]interface{})
	Error(event string, data map[string]interface{})
}

// HealthMonitor provides periodic health logging and lifecycle events
type HealthMonitor struct {
	logger    Logger
	startTime time.Time
}

func New(logger Logger) *HealthMonitor {
	return &HealthMonitor{logger: logger, startTime: time.Now()}
}

func (h *HealthMonitor) ServiceStart(version string) {
	if h.logger == nil {
		return
	}
	h.logger.Info("service_start", map[string]interface{}{
		"version": version,
		"ts":      time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *HealthMonitor) ServiceStop() {
	if h.logger == nil {
		return
	}
	h.logger.Info("service_stop", map[string]interface{}{
		"uptime_s": time.Since(h.startTime).Seconds(),
		"ts":       time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *HealthMonitor) ReportHealth(ctx context.Context) {
	if h.logger == nil {
		return
	}
	h.logger.Info("health_check", map[string]interface{}{
		"status":   "healthy",
		"uptime_s": time.Since(h.startTime).Seconds(),
	})
}
