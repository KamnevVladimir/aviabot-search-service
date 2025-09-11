package streams

import (
	"fmt"
	"testing"
	"time"
)

func TestConsumerHealthMonitor_RecordProcessing(t *testing.T) {
	monitor := NewConsumerHealthMonitor()
	
	// Записываем успешную обработку
	monitor.RecordProcessing("test-request-123", true, 100*time.Millisecond)
	
	// Проверяем метрики
	metrics := monitor.GetMetrics()
	
	if metrics.ProcessedCount != 1 {
		t.Errorf("Expected processed count 1, got %d", metrics.ProcessedCount)
	}
	
	if metrics.ErrorCount != 0 {
		t.Errorf("Expected error count 0, got %d", metrics.ErrorCount)
	}
	
	if metrics.AverageLatency != 100*time.Millisecond {
		t.Errorf("Expected average latency 100ms, got %v", metrics.AverageLatency)
	}
}

func TestConsumerHealthMonitor_RecordError(t *testing.T) {
	monitor := NewConsumerHealthMonitor()
	
	// Записываем ошибку
	monitor.RecordProcessing("test-request-123", false, 50*time.Millisecond)
	
	// Проверяем метрики
	metrics := monitor.GetMetrics()
	
	if metrics.ProcessedCount != 0 {
		t.Errorf("Expected processed count 0, got %d", metrics.ProcessedCount)
	}
	
	if metrics.ErrorCount != 1 {
		t.Errorf("Expected error count 1, got %d", metrics.ErrorCount)
	}
}

func TestConsumerHealthMonitor_AverageLatency(t *testing.T) {
	monitor := NewConsumerHealthMonitor()
	
	// Записываем несколько обработок с разной задержкой
	monitor.RecordProcessing("request-1", true, 100*time.Millisecond)
	monitor.RecordProcessing("request-2", true, 200*time.Millisecond)
	monitor.RecordProcessing("request-3", true, 300*time.Millisecond)
	
	// Проверяем среднюю задержку
	metrics := monitor.GetMetrics()
	expectedLatency := 200 * time.Millisecond
	
	if metrics.AverageLatency != expectedLatency {
		t.Errorf("Expected average latency %v, got %v", expectedLatency, metrics.AverageLatency)
	}
}

func TestConsumerHealthMonitor_Reset(t *testing.T) {
	monitor := NewConsumerHealthMonitor()
	
	// Записываем данные
	monitor.RecordProcessing("request-1", true, 100*time.Millisecond)
	monitor.RecordProcessing("request-2", false, 50*time.Millisecond)
	
	// Сбрасываем
	monitor.Reset()
	
	// Проверяем что метрики сброшены
	metrics := monitor.GetMetrics()
	
	if metrics.ProcessedCount != 0 {
		t.Errorf("Expected processed count 0, got %d", metrics.ProcessedCount)
	}
	
	if metrics.ErrorCount != 0 {
		t.Errorf("Expected error count 0, got %d", metrics.ErrorCount)
	}
	
	if metrics.AverageLatency != 0 {
		t.Errorf("Expected average latency 0, got %v", metrics.AverageLatency)
	}
}

func TestConsumerHealthMonitor_ConcurrentAccess(t *testing.T) {
	monitor := NewConsumerHealthMonitor()
	
	// Симулируем конкурентный доступ
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(i int) {
			monitor.RecordProcessing(fmt.Sprintf("request-%d", i), true, time.Duration(i*10)*time.Millisecond)
			done <- true
		}(i)
	}
	
	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Проверяем метрики
	metrics := monitor.GetMetrics()
	
	if metrics.ProcessedCount != 10 {
		t.Errorf("Expected processed count 10, got %d", metrics.ProcessedCount)
	}
}
