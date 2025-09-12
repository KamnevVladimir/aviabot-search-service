package streams

import (
	"context"
	"testing"
	"time"
)

func TestSearchRequestConsumer_Consume(t *testing.T) {
	// Создаем мок Redis клиента
	mockRedis := &mockRedisClient{
		streams:   make(map[string][]map[string]interface{}),
		processed: make(map[string]bool),
	}

	consumer := NewSearchRequestConsumer(mockRedis, "test-group")

	// Добавляем тестовое событие в stream
	event := map[string]interface{}{
		"request_id":     "test-request-123",
		"correlation_id": "test-correlation-456",
		"chat_id":        "12345",
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

	mockRedis.AddToStream("search.requests", event)

	// Тестируем консьюм
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	request, err := consumer.Consume(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if request.RequestID != "test-request-123" {
		t.Errorf("Expected request_id 'test-request-123', got %s", request.RequestID)
	}

	if request.ChatID != "12345" {
		t.Errorf("Expected chat_id '12345', got %s", request.ChatID)
	}

	if request.Params.Origin != "MOW" {
		t.Errorf("Expected origin 'MOW', got %s", request.Params.Origin)
	}

	if request.Params.Destination != "PAR" {
		t.Errorf("Expected destination 'PAR', got %s", request.Params.Destination)
	}
}

func TestSearchRequestConsumer_EmptyStream(t *testing.T) {
	mockRedis := &mockRedisClient{
		streams:   make(map[string][]map[string]interface{}),
		processed: make(map[string]bool),
	}

	consumer := NewSearchRequestConsumer(mockRedis, "test-group")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := consumer.Consume(ctx)
	if err == nil {
		t.Error("Expected timeout error for empty stream")
	}
}

func TestSearchRequestConsumer_InvalidJSON(t *testing.T) {
	mockRedis := &mockRedisClient{
		streams:   make(map[string][]map[string]interface{}),
		processed: make(map[string]bool),
	}

	consumer := NewSearchRequestConsumer(mockRedis, "test-group")

	// Добавляем невалидное событие
	event := map[string]interface{}{
		"request_id": "test-request-123",
		"params":     "invalid-json",
	}

	mockRedis.AddToStream("search.requests", event)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := consumer.Consume(ctx)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
