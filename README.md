# Search Service

Сервис поиска авиабилетов через Travelpayouts/Aviasales Data API.

## Назначение

- Поиск дешевых авиабилетов
- Форматирование сообщений с результатами поиска
- Генерация партнерских ссылок для монетизации

## Endpoints

- `GET /flights/search` - поиск билетов
- `GET /flights/message` - форматированное сообщение с результатами
- `GET /health` - проверка здоровья сервиса

## Environment Variables

- `LISTEN_ADDR` - адрес для прослушивания (по умолчанию :8084)
- `AVIASALES_TOKEN` - токен Travelpayouts API
- `AVIASALES_MARKER` - партнерский marker для ссылок
- `AVIASALES_BASE_URL` - базовый URL API (по умолчанию https://api.travelpayouts.com)
- `LOGGING_URL` - URL logging-service
- `ENVIRONMENT` - окружение (development/production)

## Локальный запуск

```bash
go run cmd/main.go
```

## Docker

```bash
docker build -t search-service .
docker run -p 8084:8084 search-service
```

## Railway Deployment

Сервис автоматически деплоится на Railway при пуше в main ветку.
