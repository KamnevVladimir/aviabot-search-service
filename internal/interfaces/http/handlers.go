package httpiface

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	app "aviasales-bot/search-service/internal/application"
)

type handler struct {
	fs     app.FlightSearcher // Новый интерфейс
	logger loggerInterface
}

// NewHandler создает новый HTTP handler с поддержкой нового интерфейса
func NewHandler(fs app.FlightSearcher) http.Handler {
	return &handler{fs: fs}
}

// loggerInterface describes minimal logger used by handlers
type loggerInterface interface {
	Info(string, map[string]interface{})
	Error(string, map[string]interface{})
}

// NewHandlerWithLogger allows injecting a logger for http handlers
func NewHandlerWithLogger(fs app.FlightSearcher, lg loggerInterface) http.Handler {
	return &handler{fs: fs, logger: lg}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	switch r.URL.Path {
	case "/flights/search":
		h.handleFlightSearch(w, r)
	case "/flights/message":
		h.handleFlightMessage(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// handleFlightSearch обрабатывает новые запросы /flights/search
func (h *handler) handleFlightSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	q := r.URL.Query()

	// Парсим параметры запроса
	p := app.SearchParams{
		Origin:      q.Get("origin"),
		Destination: q.Get("destination"),
		DepartDate:  q.Get("depart_date"),
		ReturnDate:  q.Get("return_date"),
		Currency:    coalesce(q.Get("currency"), "rub"),
		Limit:       parseIntOrDefault(q.Get("limit"), 10),
	}

	// Валидация обязательных параметров
	if p.Origin == "" || p.Destination == "" || p.DepartDate == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "origin, destination and depart_date are required",
		})
		if h.logger != nil {
			durMs := time.Since(start).Milliseconds()
			if durMs == 0 {
				durMs = 1
			}
			h.logger.Error("http_request", map[string]interface{}{
				"path":        r.URL.Path,
				"status":      http.StatusBadRequest,
				"success":     false,
				"duration_ms": durMs,
			})
		}
		return
	}

	ctx := r.Context()
	flights, err := h.fs.SearchCheap(ctx, p)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		if h.logger != nil {
			durMs := time.Since(start).Milliseconds()
			if durMs == 0 {
				durMs = 1
			}
			h.logger.Error("http_request", map[string]interface{}{
				"path":        r.URL.Path,
				"status":      http.StatusBadGateway,
				"success":     false,
				"duration_ms": durMs,
			})
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"flights": flights,
		"count":   len(flights),
	})
	if h.logger != nil {
		durMs := time.Since(start).Milliseconds()
		if durMs == 0 {
			durMs = 1
		}
		h.logger.Info("http_request", map[string]interface{}{
			"path":        r.URL.Path,
			"status":      http.StatusOK,
			"success":     true,
			"count":       len(flights),
			"duration_ms": durMs,
		})
	}
}

// handleFlightMessage обрабатывает запросы форматирования сообщений /flights/message
func (h *handler) handleFlightMessage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Парсим параметры запроса
	p := app.SearchParams{
		Origin:      q.Get("origin"),
		Destination: q.Get("destination"),
		DepartDate:  q.Get("depart_date"),
		ReturnDate:  q.Get("return_date"),
		Currency:    coalesce(q.Get("currency"), "rub"),
		Limit:       parseIntOrDefault(q.Get("limit"), 3),
	}

	originCity := coalesce(q.Get("origin_city"), p.Origin)
	destCity := coalesce(q.Get("dest_city"), p.Destination)
	passengers := parseIntOrDefault(q.Get("passengers"), 1)

	// Валидация обязательных параметров
	if p.Origin == "" || p.Destination == "" || p.DepartDate == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "origin, destination and depart_date are required",
		})
		return
	}

	ctx := r.Context()
	flights, err := h.fs.SearchCheap(ctx, p)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	message := h.fs.FormatFlightMessage(originCity, destCity, flights, passengers)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    message,
		"flights":    flights,
		"count":      len(flights),
		"passengers": passengers,
	})
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func parseIntOrDefault(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultValue
}
