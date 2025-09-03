# Примеры использования Search Service

## Настройка

Установите переменные окружения:
```bash
export AVIASALES_TOKEN=your_travelpayouts_token
export AVIASALES_MARKER=your_partner_marker
export AVIASALES_BASE_URL=https://api.travelpayouts.com
```

## Запуск сервиса

```bash
go run cmd/main.go
```

Сервис будет доступен на порту `:8084`

## Доступные endpoints

### 1. Health Check
```bash
GET /health
```

Ответ:
```json
{
  "status": "ok",
  "service": "search-service"
}
```

### 2. Legacy поиск (обратная совместимость)
```bash
GET /search?origin=MOW&destination=PAR&depart_date=2024-12&currency=rub
```

### 3. Новый поиск авиабилетов
```bash
GET /flights/search?origin=MOW&destination=PAR&depart_date=2024-12-15&return_date=2024-12-22&currency=rub&limit=5
```

Ответ:
```json
{
  "success": true,
  "flights": [
    {
      "origin": "MOW",
      "destination": "PAR",
      "depart_date": "2024-12-15T10:30:00.000Z",
      "return_date": "2024-12-22T15:45:00.000Z",
      "price": 15000,
      "airline": "SU",
      "duration": 215,
      "gate": "aviasales"
    }
  ],
  "count": 1
}
```

### 4. Форматированное сообщение с билетами
```bash
GET /flights/message?origin=MOW&destination=PAR&depart_date=2024-12-15&return_date=2024-12-22&origin_city=Москва&dest_city=Париж&passengers=2
```

Ответ:
```json
{
  "success": true,
  "message": "✈️ <b>Москва → Париж</b>\n\n🎫 <b>15 000 ₽</b>\n📅 15 дек → 22 дек\n🛫 SU • 3ч 35м\n🔗 <a href=\"https://www.aviasales.com/search/MOW1512PAR2212?marker=668475&passengers=2\">Купить билет</a>\n\n💡 <i>Цены указаны за одного пассажира в обе стороны</i>",
  "flights": [...],
  "count": 1,
  "passengers": 2
}
```

## Примеры запросов

### Поиск билетов за декабрь
```bash
curl "http://localhost:8084/flights/search?origin=MOW&destination=PAR&depart_date=2024-12&currency=rub&limit=3"
```

### Поиск билетов на точную дату
```bash
curl "http://localhost:8084/flights/search?origin=MOW&destination=PAR&depart_date=2024-12-15&return_date=2024-12-22&currency=rub"
```

### Получение готового сообщения для Telegram бота
```bash
curl "http://localhost:8084/flights/message?origin=MOW&destination=PAR&depart_date=2024-12-15&origin_city=Москва&dest_city=Париж&passengers=2"
```

## Параметры запроса

### Обязательные:
- `origin` - IATA код города отправления (MOW, LED, etc.)
- `destination` - IATA код города назначения (PAR, LON, etc.) 
- `depart_date` - Дата вылета (YYYY-MM-DD или YYYY-MM)

### Опциональные:
- `return_date` - Дата возвращения (YYYY-MM-DD или YYYY-MM)
- `currency` - Валюта (rub, usd, eur) [по умолчанию: rub]
- `limit` - Максимальное количество результатов [по умолчанию: 10]
- `passengers` - Количество пассажиров [по умолчанию: 1]
- `origin_city` - Название города отправления для сообщения
- `dest_city` - Название города назначения для сообщения

## Интеграция с Telegram ботом

Endpoint `/flights/message` возвращает готовое HTML сообщение для отправки в Telegram с:
- 📍 Маршрутом полета
- 💰 Ценами билетов  
- 📅 Датами вылета и возвращения
- ✈️ Авиакомпаниями
- ⏱️ Длительностью полета
- 🔗 Партнерскими ссылками для покупки

Сообщение можно сразу отправлять через Telegram Bot API с `parse_mode=HTML`. 