# Интеграция логирования в Go сервисы

## Обзор

Данный документ описывает стандарт интеграции структурированного логирования в Go сервисы на основе библиотеки `aviabot-shared-logging`. Этот стандарт обеспечивает единообразие, наблюдаемость и надежность логирования во всех сервисах.

## Обязательные требования

### 1. Зависимости

**Обязательно добавить в `go.mod`:**
```go
require (
    github.com/KamnevVladimir/aviabot-shared-logging v1.0.4-0.20250905085227-fa27ed2e78d0
    github.com/redis/go-redis/v9 v9.5.1
)
```

### 2. Переменные окружения

**Обязательные ENV переменные:**
```bash
# Логирование
LOG_LEVEL=info                    # debug, info, warn, error
LOG_FORMAT=json                   # json, text
SERVICE_NAME=your-service-name    # Имя сервиса для логов

# Redis (для offset store и кеширования)
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=your-password

# Grafana (опционально, для прямого логирования)
GRAFANA_URL=https://your-grafana.com
GRAFANA_TOKEN=your-token
```

**Рекомендуемые ENV переменные:**
```bash
# Retry конфигурация
MAX_RETRIES=3
RETRY_BASE_DELAY=1s
RETRY_MAX_DELAY=30s

# HTTP клиенты
HTTP_TIMEOUT=60s
TELEGRAM_API_TIMEOUT=25s
```

### 3. Инициализация логгера

**В `cmd/main.go`:**
```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/KamnevVladimir/aviabot-shared-logging/pkg/logger"
    "github.com/KamnevVladimir/aviabot-shared-logging/pkg/redis"
)

func main() {
    // Инициализация логгера
    loggerInstance, err := logger.NewLogger(
        logger.WithLevel(os.Getenv("LOG_LEVEL")),
        logger.WithFormat(os.Getenv("LOG_FORMAT")),
        logger.WithServiceName(os.Getenv("SERVICE_NAME")),
    )
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer loggerInstance.Close()

    // Инициализация Redis
    redisClient, err := redis.NewRedisClient(os.Getenv("REDIS_URL"))
    if err != nil {
        loggerInstance.Fatal("Failed to initialize Redis", map[string]interface{}{
            "error": err.Error(),
        })
    }
    defer redisClient.Close()

    // Инициализация сервиса с логгером
    service := NewService(loggerInstance, redisClient)
    
    // Graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        <-sigChan
        loggerInstance.Info("Shutdown signal received")
        cancel()
    }()

    // Запуск сервиса
    if err := service.Run(ctx); err != nil {
        loggerInstance.Fatal("Service failed", map[string]interface{}{
            "error": err.Error(),
        })
    }
}
```

## Структура логирования

### 1. Уровни логирования

**Используйте правильные уровни:**
- `DEBUG` - детальная отладочная информация
- `INFO` - нормальная работа сервиса
- `WARN` - потенциальные проблемы
- `ERROR` - ошибки, требующие внимания
- `FATAL` - критические ошибки (завершение работы)

### 2. Структура событий

**Обязательные поля для всех логов:**
```go
logger.Info("event_name", map[string]interface{}{
    "operation": "operation_name",     // Что делаем
    "duration_ms": 150,               // Время выполнения
    "success": true,                  // Успех/неудача
    "error": "error message",         // Только при ошибках
})
```

**Специальные поля по типам событий:**

**API вызовы:**
```go
logger.Info("external_api", map[string]interface{}{
    "api_name": "telegram",
    "endpoint": "getUpdates",
    "duration_ms": 1500,
    "status_code": 200,
    "updates_count": 5,
    "success": true,
})
```

**Коммуникация между сервисами:**
```go
logger.Info("service_communication", map[string]interface{}{
    "operation": "send_update",
    "target_service": "gateway-service",
    "duration_ms": 200,
    "update_id": 12345,
    "success": false,
    "error": "gateway returned status 503",
})
```

**Ошибки:**
```go
logger.Error("error_event", map[string]interface{}{
    "error": "failed to process update",
    "update_id": 12345,
    "attempt": 2,
    "max_retries": 3,
    "uptime": 3600.5,
})
```

**Health checks:**
```go
logger.Info("health_check", map[string]interface{}{
    "status": "healthy",
    "uptime": 3600.5,
    "checks": map[string]interface{}{
        "redis": "ok",
        "external_api": "ok",
    },
})
```

## Интеграция в компоненты

### 1. HTTP клиенты

**Обязательно добавлять логгер:**
```go
type HTTPClient struct {
    client *http.Client
    logger Logger
    baseURL string
}

func (c *HTTPClient) WithLogger(logger Logger) *HTTPClient {
    c.logger = logger
    return c
}

func (c *HTTPClient) DoRequest(ctx context.Context, method, endpoint string) error {
    start := time.Now()
    
    // Выполнение запроса
    resp, err := c.client.Do(req)
    duration := time.Since(start)
    
    // Логирование результата
    if c.logger != nil {
        c.logger.Info("external_api", map[string]interface{}{
            "api_name": "service_name",
            "endpoint": endpoint,
            "duration_ms": duration.Milliseconds(),
            "status_code": resp.StatusCode,
            "success": err == nil && resp.StatusCode < 400,
            "error": func() string {
                if err != nil {
                    return err.Error()
                }
                if resp.StatusCode >= 400 {
                    return fmt.Sprintf("HTTP %d", resp.StatusCode)
                }
                return ""
            }(),
        })
    }
    
    return err
}
```

### 2. Retry логика

**Обязательно логировать попытки:**
```go
func (s *Service) doWithRetry(ctx context.Context, operation func() error) error {
    var lastErr error
    
    for attempt := 1; attempt <= s.maxRetries; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }
        
        lastErr = err
        
        if s.logger != nil {
            s.logger.Warn("retry_attempt", map[string]interface{}{
                "attempt": attempt,
                "max_retries": s.maxRetries,
                "error": err.Error(),
                "next_retry_in_ms": s.getRetryDelay(attempt).Milliseconds(),
            })
        }
        
        if attempt < s.maxRetries {
            time.Sleep(s.getRetryDelay(attempt))
        }
    }
    
    if s.logger != nil {
        s.logger.Error("retry_failed", map[string]interface{}{
            "attempts": s.maxRetries,
            "error": lastErr.Error(),
        })
    }
    
    return lastErr
}
```

### 3. Health monitoring

**Обязательно реализовать:**
```go
type HealthMonitor struct {
    logger Logger
    checks map[string]HealthCheck
}

func (h *HealthMonitor) CheckHealth(ctx context.Context) error {
    start := time.Now()
    results := make(map[string]string)
    allHealthy := true
    
    for name, check := range h.checks {
        if err := check(ctx); err != nil {
            results[name] = "failed: " + err.Error()
            allHealthy = false
        } else {
            results[name] = "ok"
        }
    }
    
    duration := time.Since(start)
    
    h.logger.Info("health_check", map[string]interface{}{
        "status": func() string {
            if allHealthy {
                return "healthy"
            }
            return "unhealthy"
        }(),
        "duration_ms": duration.Milliseconds(),
        "checks": results,
    })
    
    if !allHealthy {
        return fmt.Errorf("health check failed: %v", results)
    }
    
    return nil
}
```

## Тестирование

### 1. Обязательные тесты

**Для каждого компонента с логгером:**
```go
func TestComponent_WithLogger(t *testing.T) {
    component := NewComponent()
    logger := &mockLogger{}
    
    result := component.WithLogger(logger)
    
    assert.NotNil(t, result)
    assert.Equal(t, logger, result.logger)
}

func TestComponent_LogsOnError(t *testing.T) {
    logger := &mockLogger{}
    component := NewComponent().WithLogger(logger)
    
    // Выполняем операцию, которая должна логировать ошибку
    err := component.DoOperation()
    
    assert.Error(t, err)
    assert.True(t, logger.HasErrorLog())
    assert.Contains(t, logger.LastErrorLog()["error"], "expected error")
}
```

**Mock логгер для тестов:**
```go
type mockLogger struct {
    logs []map[string]interface{}
    mu   sync.RWMutex
}

func (m *mockLogger) Info(event string, data map[string]interface{}) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.logs = append(m.logs, map[string]interface{}{
        "level": "info",
        "event": event,
        "data": data,
    })
}

func (m *mockLogger) Error(event string, data map[string]interface{}) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.logs = append(m.logs, map[string]interface{}{
        "level": "error",
        "event": event,
        "data": data,
    })
}

func (m *mockLogger) HasErrorLog() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    for _, log := range m.logs {
        if log["level"] == "error" {
            return true
        }
    }
    return false
}

func (m *mockLogger) LastErrorLog() map[string]interface{} {
    m.mu.RLock()
    defer m.mu.RUnlock()
    for i := len(m.logs) - 1; i >= 0; i-- {
        if m.logs[i]["level"] == "error" {
            return m.logs[i]["data"].(map[string]interface{})
        }
    }
    return nil
}
```

### 2. Покрытие тестами

**Обязательные требования:**
- Минимум 90% покрытие для критических пакетов (config, adapters, monitor, redis, clients)
- Минимум 70% покрытие для polling пакетов
- Исключить из покрытия: `cmd/main.go`, системную логику

**Dockerfile проверка покрытия:**
```dockerfile
# Run tests with coverage - THIS IS MANDATORY FOR DEPLOYMENT
RUN go test ./... -cover -coverprofile=coverage.out

# Verify minimum coverage for critical packages
RUN go tool cover -func=coverage.out | grep -E '(adapters|config|monitor|redis|clients)' | \
    awk 'BEGIN{total=0; count=0} {if($3+0 < 90.0) {print "Coverage too low for " $1 ": " $3; exit 1} else {print "Coverage OK for " $1 ": " $3; total+=$3; count++}} END{if(count > 0 && total/count < 90.0) {print "Average coverage too low: " total/count; exit 1} else {print "Average coverage OK: " total/count}}'

# Verify minimum coverage for polling package
RUN go tool cover -func=coverage.out | grep 'polling' | \
    awk 'BEGIN{total=0; count=0} {if($3+0 < 70.0) {print "Coverage too low for " $1 ": " $3; exit 1} else {print "Coverage OK for " $1 ": " $3; total+=$3; count++}} END{if(count > 0 && total/count < 70.0) {print "Average coverage too low: " total/count; exit 1} else {print "Average coverage OK: " total/count}}'
```

## Dockerfile требования

### 1. Обязательные этапы

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Test stage - MANDATORY
FROM builder AS test
RUN go test ./... -cover -coverprofile=coverage.out

# Coverage verification - MANDATORY
RUN go tool cover -func=coverage.out | grep -E '(adapters|config|monitor|redis|clients)' | \
    awk 'BEGIN{total=0; count=0} {if($3+0 < 90.0) {print "Coverage too low for " $1 ": " $3; exit 1} else {print "Coverage OK for " $1 ": " $3; total+=$3; count++}} END{if(count > 0 && total/count < 90.0) {print "Average coverage too low: " total/count; exit 1} else {print "Average coverage OK: " total/count}}'

# Production stage
FROM alpine:latest AS production
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/your-service .
CMD ["./your-service"]
```

### 2. Что НЕ включать в Dockerfile

**Запрещено:**
- Тесты для технических метрик логирования
- Тесты для mock объектов (только для реальной логики)
- Тесты для системных вызовов
- Тесты для main функции

## Строгие запреты

### 1. Логирование

**НЕЛЬЗЯ:**
- Использовать `log.Printf`, `fmt.Printf` для логирования
- Логировать пароли, токены, персональные данные
- Логировать без структурированных данных
- Использовать panic для обычных ошибок
- Логировать в бесконечных циклах без ограничений

**МОЖНО:**
- Использовать только `aviabot-shared-logging`
- Логировать только безопасные данные
- Использовать структурированные поля
- Использовать panic только для критических ошибок инициализации
- Логировать с ограничениями по частоте

### 2. Тестирование

**НЕЛЬЗЯ:**
- Писать тесты без проверки логов
- Использовать реальные внешние сервисы в тестах
- Игнорировать покрытие тестами
- Тестировать только happy path

**МОЖНО:**
- Использовать mock объекты для внешних зависимостей
- Тестировать error scenarios
- Проверять корректность логирования
- Использовать table-driven tests

### 3. Конфигурация

**НЕЛЬЗЯ:**
- Хардкодить значения в коде
- Использовать небезопасные значения по умолчанию
- Игнорировать ошибки инициализации

**МОЖНО:**
- Использовать переменные окружения
- Предоставлять безопасные значения по умолчанию
- Обрабатывать все ошибки инициализации

## Примеры интеграции

### 1. Простой сервис

```go
// internal/service/service.go
type Service struct {
    logger Logger
    client HTTPClient
}

func NewService(logger Logger, client HTTPClient) *Service {
    return &Service{
        logger: logger,
        client: client.WithLogger(logger),
    }
}

func (s *Service) ProcessData(ctx context.Context, data []byte) error {
    start := time.Now()
    
    s.logger.Info("processing_started", map[string]interface{}{
        "data_size": len(data),
    })
    
    result, err := s.client.Process(ctx, data)
    duration := time.Since(start)
    
    if err != nil {
        s.logger.Error("processing_failed", map[string]interface{}{
            "error": err.Error(),
            "duration_ms": duration.Milliseconds(),
        })
        return err
    }
    
    s.logger.Info("processing_completed", map[string]interface{}{
        "result_size": len(result),
        "duration_ms": duration.Milliseconds(),
    })
    
    return nil
}
```

### 2. С retry логикой

```go
func (s *Service) ProcessWithRetry(ctx context.Context, data []byte) error {
    return s.doWithRetry(ctx, func() error {
        return s.ProcessData(ctx, data)
    })
}

func (s *Service) doWithRetry(ctx context.Context, operation func() error) error {
    var lastErr error
    
    for attempt := 1; attempt <= s.maxRetries; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }
        
        lastErr = err
        
        s.logger.Warn("retry_attempt", map[string]interface{}{
            "attempt": attempt,
            "max_retries": s.maxRetries,
            "error": err.Error(),
        })
        
        if attempt < s.maxRetries {
            time.Sleep(s.getRetryDelay(attempt))
        }
    }
    
    s.logger.Error("retry_failed", map[string]interface{}{
        "attempts": s.maxRetries,
        "error": lastErr.Error(),
    })
    
    return lastErr
}
```

## Мониторинг и алерты

### 1. Ключевые метрики

**Обязательно отслеживать:**
- Количество ERROR логов в минуту
- Время ответа внешних API
- Успешность health checks
- Количество retry попыток

### 2. Настройка алертов

**Критические алерты:**
- ERROR логов > 10 в минуту
- Health check failed > 3 раза подряд
- Внешний API недоступен > 5 минут

**Предупреждающие алерты:**
- WARN логов > 50 в минуту
- Retry попыток > 20 в минуту
- Время ответа API > 5 секунд

## Заключение

Данный стандарт обеспечивает:
- Единообразие логирования во всех сервисах
- Высокую наблюдаемость и отладку
- Надежность и стабильность работы
- Простоту интеграции и поддержки

**Помните:** Логирование - это не просто вывод текста, это инструмент для понимания работы системы и быстрого решения проблем. Каждый лог должен быть полезным и структурированным.
