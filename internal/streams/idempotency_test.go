package streams

import (
	"context"
	"testing"
)

func TestIdempotencyTracker_IsProcessed(t *testing.T) {
	mockRedis := &mockRedisClient{
		processed: make(map[string]bool),
	}

	tracker := NewIdempotencyTracker(mockRedis)

	requestID := "test-request-123"

	// Первый раз - не обработан
	processed, err := tracker.IsProcessed(context.Background(), requestID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if processed {
		t.Error("Expected request to not be processed")
	}

	// Отмечаем как обработанный
	err = tracker.MarkProcessed(context.Background(), requestID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Второй раз - уже обработан
	processed, err = tracker.IsProcessed(context.Background(), requestID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !processed {
		t.Error("Expected request to be processed")
	}
}

func TestIdempotencyTracker_Expiration(t *testing.T) {
	mockRedis := &mockRedisClient{
		processed: make(map[string]bool),
	}

	tracker := NewIdempotencyTracker(mockRedis)

	requestID := "test-request-123"

	// Отмечаем как обработанный
	err := tracker.MarkProcessed(context.Background(), requestID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Проверяем что обработан
	processed, err := tracker.IsProcessed(context.Background(), requestID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !processed {
		t.Error("Expected request to be processed")
	}

	// Симулируем истечение TTL
	mockRedis.ExpireKey("processed:" + requestID)

	// Теперь не должен быть обработанным
	processed, err = tracker.IsProcessed(context.Background(), requestID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if processed {
		t.Error("Expected request to not be processed after expiration")
	}
}

func TestIdempotencyTracker_ConcurrentAccess(t *testing.T) {
	mockRedis := &mockRedisClient{
		processed: make(map[string]bool),
	}

	tracker := NewIdempotencyTracker(mockRedis)

	requestID := "test-request-123"

	// Симулируем конкурентный доступ
	done := make(chan bool, 2)

	// Горутина 1
	go func() {
		processed, _ := tracker.IsProcessed(context.Background(), requestID)
		if !processed {
			tracker.MarkProcessed(context.Background(), requestID)
		}
		done <- true
	}()

	// Горутина 2
	go func() {
		processed, _ := tracker.IsProcessed(context.Background(), requestID)
		if !processed {
			tracker.MarkProcessed(context.Background(), requestID)
		}
		done <- true
	}()

	// Ждем завершения обеих горутин
	<-done
	<-done

	// Проверяем что запрос отмечен как обработанный
	processed, err := tracker.IsProcessed(context.Background(), requestID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !processed {
		t.Error("Expected request to be processed")
	}
}
