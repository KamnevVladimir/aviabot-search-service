package streams

import (
	"context"
	"fmt"
	"time"
)

// mockRedisClient для тестирования
type mockRedisClient struct {
	streams   map[string][]map[string]interface{}
	processed map[string]bool
}

func (m *mockRedisClient) AddToStream(streamName string, event map[string]interface{}) {
	if m.streams[streamName] == nil {
		m.streams[streamName] = make([]map[string]interface{}, 0)
	}
	m.streams[streamName] = append(m.streams[streamName], event)
}

func (m *mockRedisClient) XReadGroup(ctx context.Context, group, consumer, stream string, count int64) ([]map[string]interface{}, error) {
	streams, exists := m.streams[stream]
	if !exists || len(streams) == 0 {
		return nil, nil // No data available
	}

	// Возвращаем первое событие и удаляем его
	event := streams[0]
	m.streams[stream] = streams[1:]

	return []map[string]interface{}{event}, nil
}

func (m *mockRedisClient) XAck(ctx context.Context, stream, group, messageID string) error {
	return nil
}

func (m *mockRedisClient) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.processed[key] = true
	return nil
}

func (m *mockRedisClient) Get(ctx context.Context, key string) (interface{}, error) {
	if processed, exists := m.processed[key]; exists && processed {
		return "1", nil
	}
	return nil, nil
}

func (m *mockRedisClient) GetStreams() map[string][]map[string]interface{} {
	return m.streams
}

func (m *mockRedisClient) XAdd(ctx context.Context, stream string, fields map[string]interface{}) (string, error) {
	if m.streams[stream] == nil {
		m.streams[stream] = make([]map[string]interface{}, 0)
	}

	// Генерируем mock message ID
	messageID := fmt.Sprintf("%d-0", time.Now().UnixNano())

	// Добавляем timestamp
	fields["timestamp"] = time.Now().Unix()

	m.streams[stream] = append(m.streams[stream], fields)
	return messageID, nil
}

func (m *mockRedisClient) ExpireKey(key string) {
	delete(m.processed, key)
}
