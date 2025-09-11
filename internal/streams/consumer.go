package streams

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SearchRequest представляет запрос на поиск авиабилетов из Redis Stream
type SearchRequest struct {
	RequestID     string                 `json:"request_id"`
	CorrelationID string                 `json:"correlation_id"`
	ChatID        string                 `json:"chat_id"`
	Params        SearchRequestParams    `json:"params"`
}

// SearchRequestParams параметры поиска
type SearchRequestParams struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	DepartDate  string `json:"depart_date"`
	ReturnDate  string `json:"return_date"`
	Currency    string `json:"currency"`
	Passengers  int    `json:"passengers"`
	Limit       int    `json:"limit"`
}

// RedisClient интерфейс для работы с Redis
type RedisClient interface {
	XReadGroup(ctx context.Context, group, consumer, stream string, count int64) ([]map[string]interface{}, error)
	XAck(ctx context.Context, stream, group, messageID string) error
}

// SearchRequestConsumer консьюмер для обработки запросов поиска
type SearchRequestConsumer struct {
	redis  RedisClient
	group  string
	stream string
}

// NewSearchRequestConsumer создает новый консьюмер
func NewSearchRequestConsumer(redis RedisClient, group string) *SearchRequestConsumer {
	return &SearchRequestConsumer{
		redis:  redis,
		group:  group,
		stream: "search.requests",
	}
}

// Consume читает и парсит запрос из Redis Stream
func (c *SearchRequestConsumer) Consume(ctx context.Context) (*SearchRequest, error) {
	// Читаем из stream с таймаутом
	events, err := c.redis.XReadGroup(ctx, c.group, "search-service", c.stream, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stream: %w", err)
	}
	
	if len(events) == 0 {
		return nil, fmt.Errorf("no events available")
	}
	
	event := events[0]
	
	// Парсим JSON из поля "params"
	paramsJSON, exists := event["params"]
	if !exists {
		return nil, fmt.Errorf("missing params field")
	}
	
	paramsBytes, err := json.Marshal(paramsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}
	
	var params SearchRequestParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}
	
	// Создаем SearchRequest
	request := &SearchRequest{
		RequestID:     getString(event, "request_id"),
		CorrelationID: getString(event, "correlation_id"),
		ChatID:        getString(event, "chat_id"),
		Params:        params,
	}
	
	// Валидируем обязательные поля
	if request.RequestID == "" {
		return nil, fmt.Errorf("missing request_id")
	}
	if request.ChatID == "" {
		return nil, fmt.Errorf("missing chat_id")
	}
	if request.Params.Origin == "" {
		return nil, fmt.Errorf("missing origin")
	}
	if request.Params.Destination == "" {
		return nil, fmt.Errorf("missing destination")
	}
	
	return request, nil
}

// ConsumeWithTimeout читает запрос с таймаутом
func (c *SearchRequestConsumer) ConsumeWithTimeout(ctx context.Context, timeout time.Duration) (*SearchRequest, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	return c.Consume(ctx)
}

// getString извлекает строковое значение из map
func getString(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
