package httpiface

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	app "aviasales-bot/search-service/internal/application"
)

type incomingRequestTestLogger struct {
	events []logEvent
}

type logEvent struct {
	level string
	event string
	data  map[string]interface{}
}

func (t *incomingRequestTestLogger) Info(event string, data map[string]interface{}) {
	t.events = append(t.events, logEvent{"INFO", event, data})
}

func (t *incomingRequestTestLogger) Error(event string, data map[string]interface{}) {
	t.events = append(t.events, logEvent{"ERROR", event, data})
}

func (t *incomingRequestTestLogger) getEvents() []logEvent {
	return t.events
}

func (t *incomingRequestTestLogger) clear() {
	t.events = nil
}

type incomingRequestMockFlightSearcher struct{}

func (m *incomingRequestMockFlightSearcher) SearchCheap(_ context.Context, p app.SearchParams) ([]app.Flight, error) {
	return []app.Flight{
		{
			Origin:      p.Origin,
			Destination: p.Destination,
			DepartDate:  time.Now(),
			ReturnDate:  time.Now().Add(24 * time.Hour),
			Price:       1000,
			Airline:     "SU",
			Duration:    120,
		},
	}, nil
}

func (m *incomingRequestMockFlightSearcher) GeneratePartnerLink(flight app.Flight, passengers int) string {
	return "https://test.com"
}

func (m *incomingRequestMockFlightSearcher) FormatFlightMessage(originCity, destCity string, flights []app.Flight, passengers int) string {
	return "Test message"
}

func TestServeHTTP_LogsIncomingRequest(t *testing.T) {
	logger := &incomingRequestTestLogger{}
	handler := NewHandlerWithLogger(&incomingRequestMockFlightSearcher{}, logger)

	req := httptest.NewRequest("GET", "/flights/search?origin=LED&destination=MOW&depart_date=2024-01-01", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	events := logger.getEvents()
	if len(events) == 0 {
		t.Fatal("Expected at least one log event")
	}

	// Проверяем, что есть событие входящего запроса
	var incomingRequestFound bool
	for _, event := range events {
		if event.event == "http_request" {
			incomingRequestFound = true
			if event.data["path"] != "/flights/search" {
				t.Errorf("Expected path '/flights/search', got %v", event.data["path"])
			}
			if event.data["method"] != "GET" {
				t.Errorf("Expected method 'GET', got %v", event.data["method"])
			}
			if event.data["remote_addr"] == nil {
				t.Error("Expected remote_addr to be set")
			}
			if event.data["user_agent"] == nil {
				t.Error("Expected user_agent to be set")
			}
			break
		}
	}

	if !incomingRequestFound {
		t.Error("Expected incoming request log event not found")
	}
}

func TestServeHTTP_LogsIncomingRequestForMessageEndpoint(t *testing.T) {
	logger := &incomingRequestTestLogger{}
	handler := NewHandlerWithLogger(&incomingRequestMockFlightSearcher{}, logger)

	req := httptest.NewRequest("GET", "/flights/message?origin=LED&destination=MOW&depart_date=2024-01-01", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	events := logger.getEvents()
	if len(events) == 0 {
		t.Fatal("Expected at least one log event")
	}

	// Проверяем, что есть событие входящего запроса
	var incomingRequestFound bool
	for _, event := range events {
		if event.event == "http_request" {
			incomingRequestFound = true
			if event.data["path"] != "/flights/message" {
				t.Errorf("Expected path '/flights/message', got %v", event.data["path"])
			}
			if event.data["method"] != "GET" {
				t.Errorf("Expected method 'GET', got %v", event.data["method"])
			}
			break
		}
	}

	if !incomingRequestFound {
		t.Error("Expected incoming request log event not found")
	}
}

func TestHandleFlightMessage_LogsSuccessAndDuration(t *testing.T) {
	logger := &incomingRequestTestLogger{}
	handler := NewHandlerWithLogger(&incomingRequestMockFlightSearcher{}, logger)

	req := httptest.NewRequest("GET", "/flights/message?origin=LED&destination=MOW&depart_date=2024-01-01", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	events := logger.getEvents()
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 log events, got %d", len(events))
	}

	// Проверяем, что есть событие успешного ответа
	var successEventFound bool
	for _, event := range events {
		if event.event == "http_request" && event.data["success"] == true {
			successEventFound = true
			if event.data["path"] != "/flights/message" {
				t.Errorf("Expected path '/flights/message', got %v", event.data["path"])
			}
			if event.data["status"] != 200 {
				t.Errorf("Expected status 200, got %v", event.data["status"])
			}
			if event.data["count"] != 1 {
				t.Errorf("Expected count 1, got %v", event.data["count"])
			}
			if event.data["duration_ms"] == nil {
				t.Error("Expected duration_ms to be set")
			}
			duration, ok := event.data["duration_ms"].(int64)
			if !ok || duration <= 0 {
				t.Errorf("Expected positive duration_ms, got %v", event.data["duration_ms"])
			}
			break
		}
	}

	if !successEventFound {
		t.Error("Expected success log event not found")
	}
}

func TestHandleFlightMessage_LogsBadRequest(t *testing.T) {
	logger := &incomingRequestTestLogger{}
	handler := NewHandlerWithLogger(&incomingRequestMockFlightSearcher{}, logger)

	req := httptest.NewRequest("GET", "/flights/message?origin=&destination=MOW&depart_date=2024-01-01", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	events := logger.getEvents()
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 log events, got %d", len(events))
	}

	// Проверяем, что есть событие ошибки валидации
	var errorEventFound bool
	for _, event := range events {
		if event.event == "http_request" && event.data["success"] == false {
			errorEventFound = true
			if event.data["path"] != "/flights/message" {
				t.Errorf("Expected path '/flights/message', got %v", event.data["path"])
			}
			if event.data["status"] != 400 {
				t.Errorf("Expected status 400, got %v", event.data["status"])
			}
			break
		}
	}

	if !errorEventFound {
		t.Error("Expected error log event not found")
	}
}

func TestHandleFlightMessage_LogsUpstreamError(t *testing.T) {
	logger := &incomingRequestTestLogger{}
	mockSearcher := &incomingRequestMockFlightSearcherWithError{}
	handler := NewHandlerWithLogger(mockSearcher, logger)

	req := httptest.NewRequest("GET", "/flights/message?origin=LED&destination=MOW&depart_date=2024-01-01", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	events := logger.getEvents()
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 log events, got %d", len(events))
	}

	// Проверяем, что есть событие ошибки upstream
	var errorEventFound bool
	for _, event := range events {
		if event.event == "http_request" && event.data["success"] == false {
			errorEventFound = true
			if event.data["path"] != "/flights/message" {
				t.Errorf("Expected path '/flights/message', got %v", event.data["path"])
			}
			if event.data["status"] != 502 {
				t.Errorf("Expected status 502, got %v", event.data["status"])
			}
			break
		}
	}

	if !errorEventFound {
		t.Error("Expected error log event not found")
	}
}

type incomingRequestMockFlightSearcherWithError struct{}

func (m *incomingRequestMockFlightSearcherWithError) SearchCheap(_ context.Context, p app.SearchParams) ([]app.Flight, error) {
	return nil, errors.New("upstream error")
}

func (m *incomingRequestMockFlightSearcherWithError) GeneratePartnerLink(flight app.Flight, passengers int) string {
	return "https://test.com"
}

func (m *incomingRequestMockFlightSearcherWithError) FormatFlightMessage(originCity, destCity string, flights []app.Flight, passengers int) string {
	return "Test message"
}
