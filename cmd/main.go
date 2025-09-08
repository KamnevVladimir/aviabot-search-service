package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	app "aviasales-bot/search-service/internal/application"
	api "aviasales-bot/search-service/internal/infrastructure/aviasales"
	httpiface "aviasales-bot/search-service/internal/interfaces/http"
	"aviasales-bot/search-service/internal/monitor"
	obslogger "aviasales-bot/search-service/internal/observability/logger"

	shared "github.com/KamnevVladimir/aviabot-shared-logging"
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

	// init shared logging client if LOGGING_URL provided
	var lg obslogger.Logger = obslogger.NoopLogger{}
	if loggingURL := os.Getenv("LOGGING_URL"); loggingURL != "" {
		c := shared.NewClient(loggingURL, "search-service")
		lg = obslogger.NewSharedAdapter(c)
		defer lg.Close()
	}

	// health monitor
	hm := monitor.New(lg)
	hm.ServiceStart("v1.0.0")

	client := api.NewClient(baseURL, token, marker, api.WithLogger(lg))

	adapter := &clientAdapter{c: client}

	h := httpiface.NewHandlerWithLogger(adapter, convertLogger(lg))

	// Routing
	http.Handle("/", h)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"search-service"}`))
	})

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8084"
	}

	// periodic health reporting
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				hm.ReportHealth(ctx)
			}
		}
	}()

	// graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		hm.ServiceStop()
		cancel()
		os.Exit(0)
	}()

	lg.Info("service_start", map[string]interface{}{"addr": addr, "ts": time.Now().UTC().Format(time.RFC3339)})
	log.Fatal(http.ListenAndServe(addr, nil))
}

// clientAdapter адаптер который реализует FlightSearcher интерфейс
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

// convertLogger adapts observability logger to handler's minimal interface
func convertLogger(l obslogger.Logger) interface {
	Info(string, map[string]interface{})
	Error(string, map[string]interface{})
} {
	return &handlerLoggerAdapter{l: l}
}

type handlerLoggerAdapter struct{ l obslogger.Logger }

func (h *handlerLoggerAdapter) Info(e string, d map[string]interface{})  { h.l.Info(e, d) }
func (h *handlerLoggerAdapter) Error(e string, d map[string]interface{}) { h.l.Error(e, d) }
