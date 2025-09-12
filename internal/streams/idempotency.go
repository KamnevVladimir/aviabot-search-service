package streams

import (
	"context"
	"fmt"
	"time"
)

// IdempotencyTracker отслеживает обработанные запросы для предотвращения дублирования
type IdempotencyTracker struct {
	redis RedisIdempotency
	ttl   time.Duration
}

// RedisIdempotency интерфейс для работы с Redis для идемпотентности
type RedisIdempotency interface {
	SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (interface{}, error)
}

// NewIdempotencyTracker создает новый трекер идемпотентности
func NewIdempotencyTracker(redis RedisIdempotency) *IdempotencyTracker {
	return &IdempotencyTracker{
		redis: redis,
		ttl:   24 * time.Hour, // TTL 24 часа
	}
}

// IsProcessed проверяет, был ли запрос уже обработан
func (t *IdempotencyTracker) IsProcessed(ctx context.Context, requestID string) (bool, error) {
	key := fmt.Sprintf("processed:%s", requestID)

	value, err := t.redis.Get(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check processed status: %w", err)
	}

	return value != nil, nil
}

// MarkProcessed отмечает запрос как обработанный
func (t *IdempotencyTracker) MarkProcessed(ctx context.Context, requestID string) error {
	key := fmt.Sprintf("processed:%s", requestID)

	err := t.redis.SetWithTTL(ctx, key, "1", t.ttl)
	if err != nil {
		return fmt.Errorf("failed to mark as processed: %w", err)
	}

	return nil
}

// ProcessWithIdempotency выполняет обработку с проверкой идемпотентности
func (t *IdempotencyTracker) ProcessWithIdempotency(ctx context.Context, requestID string, processor func() error) error {
	// Проверяем, был ли уже обработан
	processed, err := t.IsProcessed(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to check idempotency: %w", err)
	}

	if processed {
		return nil // Уже обработан, пропускаем
	}

	// Выполняем обработку
	if err := processor(); err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	// Отмечаем как обработанный
	if err := t.MarkProcessed(ctx, requestID); err != nil {
		return fmt.Errorf("failed to mark as processed: %w", err)
	}

	return nil
}
