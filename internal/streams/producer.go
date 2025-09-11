package streams

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// FlightResult представляет результат поиска одного рейса
type FlightResult struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	DepartDate  string `json:"depart_date"`
	ReturnDate  string `json:"return_date"`
	Price       int    `json:"price"`
	Currency    string `json:"currency"`
	Link        string `json:"link"`
}

// SearchResult представляет результат поиска авиабилетов
type SearchResult struct {
	RequestID     string        `json:"request_id"`
	CorrelationID string        `json:"correlation_id"`
	ChatID        string        `json:"chat_id"`
	Count         int           `json:"count"`
	Results       []FlightResult `json:"results"`
	Error         string        `json:"error,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
}

// RedisProducer интерфейс для публикации в Redis Stream
type RedisProducer interface {
	XAdd(ctx context.Context, stream string, fields map[string]interface{}) (string, error)
}

// SearchResultProducer продюсер для публикации результатов поиска
type SearchResultProducer struct {
	redis  RedisProducer
	stream string
}

// NewSearchResultProducer создает новый продюсер
func NewSearchResultProducer(redis RedisProducer) *SearchResultProducer {
	return &SearchResultProducer{
		redis:  redis,
		stream: "search.results",
	}
}

// Publish публикует результат поиска в Redis Stream
func (p *SearchResultProducer) Publish(ctx context.Context, result *SearchResult) (string, error) {
	// Устанавливаем timestamp если не установлен
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}
	
	// Конвертируем в map для Redis
	fields := map[string]interface{}{
		"request_id":     result.RequestID,
		"correlation_id": result.CorrelationID,
		"chat_id":        result.ChatID,
		"count":          result.Count,
		"timestamp":      result.Timestamp.Unix(),
	}
	
	// Добавляем результаты или ошибку
	if result.Error != "" {
		fields["error"] = result.Error
		fields["results"] = "[]"
	} else {
		// Сериализуем результаты в JSON
		resultsJSON, err := json.Marshal(result.Results)
		if err != nil {
			return "", fmt.Errorf("failed to marshal results: %w", err)
		}
		fields["results"] = string(resultsJSON)
	}
	
	// Публикуем в Redis Stream
	messageID, err := p.redis.XAdd(ctx, p.stream, fields)
	if err != nil {
		return "", fmt.Errorf("failed to publish to stream: %w", err)
	}
	
	return messageID, nil
}

// PublishSuccess публикует успешный результат поиска
func (p *SearchResultProducer) PublishSuccess(ctx context.Context, requestID, correlationID, chatID string, results []FlightResult) (string, error) {
	result := &SearchResult{
		RequestID:     requestID,
		CorrelationID: correlationID,
		ChatID:        chatID,
		Count:         len(results),
		Results:       results,
		Timestamp:     time.Now(),
	}
	
	return p.Publish(ctx, result)
}

// PublishError публикует ошибку поиска
func (p *SearchResultProducer) PublishError(ctx context.Context, requestID, correlationID, chatID, errorMsg string) (string, error) {
	result := &SearchResult{
		RequestID:     requestID,
		CorrelationID: correlationID,
		ChatID:        chatID,
		Count:         0,
		Results:       []FlightResult{},
		Error:         errorMsg,
		Timestamp:     time.Now(),
	}
	
	return p.Publish(ctx, result)
}
