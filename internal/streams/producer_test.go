package streams

import (
	"context"
	"testing"
)

func TestSearchResultProducer_PublishSuccess(t *testing.T) {
	mockRedis := &mockRedisClient{
		streams:   make(map[string][]map[string]interface{}),
		processed: make(map[string]bool),
	}
	
	producer := NewSearchResultProducer(mockRedis)
	
	result := &SearchResult{
		RequestID:     "test-request-123",
		CorrelationID: "test-correlation-456",
		ChatID:        "12345",
		Count:         2,
		Results: []FlightResult{
			{
				Origin:      "MOW",
				Destination: "PAR",
				DepartDate:  "2024-12-15",
				ReturnDate:  "2024-12-22",
				Price:       12345,
				Currency:    "rub",
				Link:        "https://example.com/flight1",
			},
			{
				Origin:      "MOW",
				Destination: "PAR",
				DepartDate:  "2024-12-16",
				ReturnDate:  "2024-12-23",
				Price:       13456,
				Currency:    "rub",
				Link:        "https://example.com/flight2",
			},
		},
	}
	
	ctx := context.Background()
	messageID, err := producer.Publish(ctx, result)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if messageID == "" {
		t.Error("Expected non-empty message ID")
	}
	
	// Проверяем что событие добавлено в stream
	streams := mockRedis.GetStreams()
	if len(streams["search.results"]) == 0 {
		t.Error("Expected event to be added to search.results stream")
	}
}

func TestSearchResultProducer_PublishError(t *testing.T) {
	mockRedis := &mockRedisClient{
		streams:   make(map[string][]map[string]interface{}),
		processed: make(map[string]bool),
	}
	
	producer := NewSearchResultProducer(mockRedis)
	
	result := &SearchResult{
		RequestID:     "test-request-123",
		CorrelationID: "test-correlation-456",
		ChatID:        "12345",
		Count:         0,
		Error:         "API timeout",
		Results:       []FlightResult{},
	}
	
	ctx := context.Background()
	messageID, err := producer.Publish(ctx, result)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if messageID == "" {
		t.Error("Expected non-empty message ID")
	}
	
	// Проверяем что событие добавлено в stream
	streams := mockRedis.GetStreams()
	if len(streams["search.results"]) == 0 {
		t.Error("Expected event to be added to search.results stream")
	}
}

func TestSearchResultProducer_EmptyResult(t *testing.T) {
	mockRedis := &mockRedisClient{
		streams:   make(map[string][]map[string]interface{}),
		processed: make(map[string]bool),
	}
	
	producer := NewSearchResultProducer(mockRedis)
	
	result := &SearchResult{
		RequestID:     "test-request-123",
		CorrelationID: "test-correlation-456",
		ChatID:        "12345",
		Count:         0,
		Results:       []FlightResult{},
	}
	
	ctx := context.Background()
	messageID, err := producer.Publish(ctx, result)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if messageID == "" {
		t.Error("Expected non-empty message ID")
	}
}

