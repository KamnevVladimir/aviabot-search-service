package streams

import (
	"context"
	"testing"
	"time"
)

func TestRedisStreamsIntegration(t *testing.T) {
	// Создаем мок Redis клиент
	mockRedis := &mockRedisClient{
		streams:   make(map[string][]map[string]interface{}),
		processed: make(map[string]bool),
	}
	
	// Создаем компоненты
	consumer := NewSearchRequestConsumer(mockRedis, "test-group")
	producer := NewSearchResultProducer(mockRedis)
	tracker := NewIdempotencyTracker(mockRedis)
	monitor := NewConsumerHealthMonitor()
	
	// Тестовый запрос
	requestID := "test-request-123"
	correlationID := "test-correlation-456"
	chatID := "12345"
	
	// Добавляем запрос в stream
	requestEvent := map[string]interface{}{
		"request_id":     requestID,
		"correlation_id": correlationID,
		"chat_id":        chatID,
		"params": map[string]interface{}{
			"origin":      "MOW",
			"destination": "PAR",
			"depart_date": "2024-12-15",
			"return_date": "2024-12-22",
			"currency":    "rub",
			"passengers":  1,
			"limit":       5,
		},
	}
	
	mockRedis.AddToStream("search.requests", requestEvent)
	
	// Тестируем полный цикл обработки
	ctx := context.Background()
	
	// 1. Читаем запрос
	request, err := consumer.Consume(ctx)
	if err != nil {
		t.Fatalf("Failed to consume request: %v", err)
	}
	
	if request.RequestID != requestID {
		t.Errorf("Expected request_id %s, got %s", requestID, request.RequestID)
	}
	
	// 2. Проверяем идемпотентность
	processed, err := tracker.IsProcessed(ctx, requestID)
	if err != nil {
		t.Fatalf("Failed to check idempotency: %v", err)
	}
	
	if processed {
		t.Error("Request should not be processed yet")
	}
	
	// 3. Отмечаем как обработанный
	err = tracker.MarkProcessed(ctx, requestID)
	if err != nil {
		t.Fatalf("Failed to mark as processed: %v", err)
	}
	
	// 4. Записываем метрики
	start := time.Now()
	monitor.RecordProcessing(requestID, true, time.Since(start))
	
	// 5. Публикуем результат
	results := []FlightResult{
		{
			Origin:      "MOW",
			Destination: "PAR",
			DepartDate:  "2024-12-15",
			ReturnDate:  "2024-12-22",
			Price:       12345,
			Currency:    "rub",
			Link:        "https://example.com/flight1",
		},
	}
	
	messageID, err := producer.PublishSuccess(ctx, requestID, correlationID, chatID, results)
	if err != nil {
		t.Fatalf("Failed to publish result: %v", err)
	}
	
	if messageID == "" {
		t.Error("Expected non-empty message ID")
	}
	
	// 6. Проверяем что результат опубликован
	streams := mockRedis.GetStreams()
	if len(streams["search.results"]) == 0 {
		t.Error("Expected result to be published to search.results stream")
	}
	
	// 7. Проверяем метрики
	metrics := monitor.GetMetrics()
	if metrics.ProcessedCount != 1 {
		t.Errorf("Expected processed count 1, got %d", metrics.ProcessedCount)
	}
	
	if !monitor.IsHealthy() {
		t.Error("Consumer should be healthy")
	}
	
	// 8. Проверяем идемпотентность повторно
	processed, err = tracker.IsProcessed(ctx, requestID)
	if err != nil {
		t.Fatalf("Failed to check idempotency: %v", err)
	}
	
	if !processed {
		t.Error("Request should be processed now")
	}
}

func TestRedisStreamsIntegration_ErrorHandling(t *testing.T) {
	mockRedis := &mockRedisClient{
		streams:   make(map[string][]map[string]interface{}),
		processed: make(map[string]bool),
	}
	
	producer := NewSearchResultProducer(mockRedis)
	monitor := NewConsumerHealthMonitor()
	
	requestID := "test-request-error"
	correlationID := "test-correlation-error"
	chatID := "12345"
	
	ctx := context.Background()
	
	// Тестируем обработку ошибки
	start := time.Now()
	monitor.RecordProcessing(requestID, false, time.Since(start))
	
	// Публикуем ошибку
	messageID, err := producer.PublishError(ctx, requestID, correlationID, chatID, "API timeout")
	if err != nil {
		t.Fatalf("Failed to publish error: %v", err)
	}
	
	if messageID == "" {
		t.Error("Expected non-empty message ID")
	}
	
	// Проверяем метрики
	metrics := monitor.GetMetrics()
	if metrics.ErrorCount != 1 {
		t.Errorf("Expected error count 1, got %d", metrics.ErrorCount)
	}
	
	if metrics.SuccessRate != 0.0 {
		t.Errorf("Expected success rate 0.0, got %f", metrics.SuccessRate)
	}
}
