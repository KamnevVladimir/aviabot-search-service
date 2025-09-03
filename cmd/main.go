package main

import (
	"context"
	"log"
	"net/http"
	"os"

	app "aviasales-bot/search-service/internal/application"
	api "aviasales-bot/search-service/internal/infrastructure/aviasales"
	httpiface "aviasales-bot/search-service/internal/interfaces/http"
)

func main() {
	token := os.Getenv("AVIASALES_TOKEN")
	if token == "" {
		log.Fatal("AVIASALES_TOKEN is required")
	}
	marker := os.Getenv("AVIASALES_MARKER")
	if marker == "" {
		marker = "668475"
	}
	baseURL := os.Getenv("AVIASALES_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.travelpayouts.com"
	}

	client := api.NewClient(baseURL, token, marker)

	// Создаем адаптер который реализует оба интерфейса
	adapter := &clientAdapter{c: client}

	// Создаем handler с новым интерфейсом
	h := httpiface.NewHandler(adapter)

	// Настраиваем роутинг для всех endpoints
	http.Handle("/", h) // Обрабатываем все запросы через один handler

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"search-service"}`))
	})

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8084"
	}

	log.Printf("🚀 search-service listening on %s", addr)
	log.Printf("📍 Available endpoints:")
	log.Printf("   GET /search - Legacy search endpoint")
	log.Printf("   GET /flights/search - New flight search")
	log.Printf("   GET /flights/message - Formatted flight messages")
	log.Printf("   GET /health - Health check")

	log.Fatal(http.ListenAndServe(addr, nil))
}

// clientAdapter адаптер который реализует оба интерфейса
type clientAdapter struct{ c *api.Client }

// Реализация нового FlightSearcher интерфейса
func (a *clientAdapter) SearchCheap(ctx context.Context, p app.SearchParams) ([]app.Flight, error) {
	// Конвертируем app.SearchParams в api.SearchParams
	apiParams := api.SearchParams{
		Origin:      p.Origin,
		Destination: p.Destination,
		DepartDate:  p.DepartDate,
		ReturnDate:  p.ReturnDate,
		Currency:    p.Currency,
		Limit:       p.Limit,
	}

	// Вызываем API и получаем результат
	flights, err := a.c.SearchCheap(ctx, apiParams)
	if err != nil {
		return nil, err
	}

	// Конвертируем api.Flight в app.Flight
	var appFlights []app.Flight
	for _, flight := range flights {
		appFlights = append(appFlights, app.Flight{
			Origin:       flight.Origin,
			Destination:  flight.Destination,
			DepartDate:   flight.DepartDate,
			ReturnDate:   flight.ReturnDate,
			Price:        flight.Price,
			Airline:      flight.Airline,
			FlightNumber: flight.FlightNumber,
			Duration:     flight.Duration,
			Distance:     flight.Distance,
			Gate:         flight.Gate,
			ExpiresAt:    flight.ExpiresAt,
			Actual:       flight.Actual,
		})
	}

	return appFlights, nil
}

func (a *clientAdapter) GeneratePartnerLink(flight app.Flight, passengers int) string {
	// Конвертируем app.Flight в api.Flight
	apiFlight := api.Flight{
		Origin:       flight.Origin,
		Destination:  flight.Destination,
		DepartDate:   flight.DepartDate,
		ReturnDate:   flight.ReturnDate,
		Price:        flight.Price,
		Airline:      flight.Airline,
		FlightNumber: flight.FlightNumber,
		Duration:     flight.Duration,
		Distance:     flight.Distance,
		Gate:         flight.Gate,
		ExpiresAt:    flight.ExpiresAt,
		Actual:       flight.Actual,
	}

	return a.c.GeneratePartnerLink(apiFlight, passengers)
}

func (a *clientAdapter) FormatFlightMessage(originCity, destCity string, flights []app.Flight, passengers int) string {
	// Конвертируем app.Flight в api.Flight
	var apiFlights []api.Flight
	for _, flight := range flights {
		apiFlights = append(apiFlights, api.Flight{
			Origin:       flight.Origin,
			Destination:  flight.Destination,
			DepartDate:   flight.DepartDate,
			ReturnDate:   flight.ReturnDate,
			Price:        flight.Price,
			Airline:      flight.Airline,
			FlightNumber: flight.FlightNumber,
			Duration:     flight.Duration,
			Distance:     flight.Distance,
			Gate:         flight.Gate,
			ExpiresAt:    flight.ExpiresAt,
			Actual:       flight.Actual,
		})
	}

	return a.c.FormatFlightMessage(originCity, destCity, apiFlights, passengers)
}

// Реализация legacy Searcher интерфейса для обратной совместимости
func (a *clientAdapter) Search(ctx context.Context, p app.SearchParams) ([]map[string]any, error) {
	// Конвертируем в api.SearchParams с поддержкой старого формата Month
	apiParams := api.SearchParams{
		Origin:      p.Origin,
		Destination: p.Destination,
		DepartDate:  p.DepartDate,
		ReturnDate:  p.ReturnDate,
		Currency:    p.Currency,
		Limit:       p.Limit,
	}

	return a.c.Search(ctx, apiParams)
}
