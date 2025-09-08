package application

import (
	"context"
	"time"
)

// SearchParams параметры поиска авиабилетов
type SearchParams struct {
	Origin      string // IATA код города отправления
	Destination string // IATA код города назначения
	DepartDate  string // Дата вылета (YYYY-MM-DD или YYYY-MM)
	ReturnDate  string // Дата возвращения (YYYY-MM-DD или YYYY-MM)
	Currency    string // Валюта (rub, usd, eur)
	Limit       int    // Максимальное количество результатов
}

// Flight представляет информацию о рейсе
type Flight struct {
	Origin       string    `json:"origin"`
	Destination  string    `json:"destination"`
	DepartDate   time.Time `json:"depart_date"`
	ReturnDate   time.Time `json:"return_date"`
	Price        int       `json:"price"`
	Airline      string    `json:"airline"`
	FlightNumber int       `json:"flight_number"`
	Duration     int       `json:"duration"`
	Distance     int       `json:"distance"`
	Gate         string    `json:"gate"`
	ExpiresAt    time.Time `json:"expires_at"`
	Actual       bool      `json:"actual"`
}

// FlightSearcher интерфейс для поиска авиабилетов
type FlightSearcher interface {
	// SearchCheap ищет самые дешевые билеты
	SearchCheap(ctx context.Context, p SearchParams) ([]Flight, error)

	// GeneratePartnerLink генерирует партнерскую ссылку для покупки
	GeneratePartnerLink(flight Flight, passengers int) string

	// FormatFlightMessage форматирует сообщение с билетами для пользователя
	FormatFlightMessage(originCity, destCity string, flights []Flight, passengers int) string
}
