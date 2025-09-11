package streams

import (
	"sync"
	"time"
)

// ConsumerHealthMonitor отслеживает здоровье консьюмера
type ConsumerHealthMonitor struct {
	mu             sync.RWMutex
	processedCount int64
	errorCount     int64
	totalLatency   time.Duration
	requestCount   int64
}

// ConsumerMetrics метрики консьюмера
type ConsumerMetrics struct {
	ProcessedCount int64         `json:"processed_count"`
	ErrorCount     int64         `json:"error_count"`
	AverageLatency time.Duration `json:"average_latency_ms"`
	SuccessRate    float64       `json:"success_rate"`
}

// NewConsumerHealthMonitor создает новый монитор здоровья
func NewConsumerHealthMonitor() *ConsumerHealthMonitor {
	return &ConsumerHealthMonitor{}
}

// RecordProcessing записывает результат обработки запроса
func (m *ConsumerHealthMonitor) RecordProcessing(requestID string, success bool, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.requestCount++
	m.totalLatency += latency
	
	if success {
		m.processedCount++
	} else {
		m.errorCount++
	}
}

// GetMetrics возвращает текущие метрики
func (m *ConsumerHealthMonitor) GetMetrics() ConsumerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var averageLatency time.Duration
	if m.requestCount > 0 {
		averageLatency = m.totalLatency / time.Duration(m.requestCount)
	}
	
	var successRate float64
	if m.requestCount > 0 {
		successRate = float64(m.processedCount) / float64(m.requestCount) * 100
	}
	
	return ConsumerMetrics{
		ProcessedCount: m.processedCount,
		ErrorCount:     m.errorCount,
		AverageLatency: averageLatency,
		SuccessRate:    successRate,
	}
}

// Reset сбрасывает все метрики
func (m *ConsumerHealthMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.processedCount = 0
	m.errorCount = 0
	m.totalLatency = 0
	m.requestCount = 0
}

// IsHealthy проверяет, здоров ли консьюмер
func (m *ConsumerHealthMonitor) IsHealthy() bool {
	metrics := m.GetMetrics()
	
	// Консьюмер считается здоровым если:
	// 1. Обработал хотя бы один запрос
	// 2. Успешность > 80%
	// 3. Средняя задержка < 5 секунд
	
	if metrics.ProcessedCount == 0 && metrics.ErrorCount == 0 {
		return true // Нет данных - считаем здоровым
	}
	
	if metrics.SuccessRate < 80.0 {
		return false
	}
	
	if metrics.AverageLatency > 5*time.Second {
		return false
	}
	
	return true
}

// GetHealthStatus возвращает статус здоровья
func (m *ConsumerHealthMonitor) GetHealthStatus() map[string]interface{} {
	metrics := m.GetMetrics()
	
	return map[string]interface{}{
		"healthy":         m.IsHealthy(),
		"processed_count": metrics.ProcessedCount,
		"error_count":     metrics.ErrorCount,
		"success_rate":    metrics.SuccessRate,
		"average_latency": metrics.AverageLatency.Milliseconds(),
	}
}
