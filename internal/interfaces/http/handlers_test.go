package httpiface

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	app "aviasales-bot/search-service/internal/application"
)

// mockFlightSearcher реализует FlightSearcher интерфейс для тестов
type mockFlightSearcher struct {
	calledWith  app.SearchParams
	shouldError bool
	mockFlights []app.Flight
	mockMessage string
	mockLink    string
}

func (m *mockFlightSearcher) SearchCheap(_ context.Context, p app.SearchParams) ([]app.Flight, error) {
	m.calledWith = p
	if m.shouldError {
		return nil, &mockError{"search failed"}
	}
	if len(m.mockFlights) > 0 {
		return m.mockFlights, nil
	}
	// Возвращаем тестовые данные по умолчанию
	return []app.Flight{
		{
			Origin:      "MOW",
			Destination: "PAR",
			DepartDate:  time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC),
			ReturnDate:  time.Date(2024, 12, 22, 15, 45, 0, 0, time.UTC),
			Price:       15000,
			Airline:     "SU",
			Duration:    215,
		},
		{
			Origin:      "MOW",
			Destination: "PAR",
			DepartDate:  time.Date(2024, 12, 20, 8, 15, 0, 0, time.UTC),
			ReturnDate:  time.Date(2024, 12, 27, 19, 30, 0, 0, time.UTC),
			Price:       17500,
			Airline:     "AF",
			Duration:    220,
		},
	}, nil
}

func (m *mockFlightSearcher) GeneratePartnerLink(flight app.Flight, passengers int) string {
	if m.mockLink != "" {
		return m.mockLink
	}
	return "https://www.aviasales.com/search/MOW1512PAR2212?marker=668475&passengers=2"
}

func (m *mockFlightSearcher) FormatFlightMessage(originCity, destCity string, flights []app.Flight, passengers int) string {
	if m.mockMessage != "" {
		return m.mockMessage
	}
	return "✈️ <b>Москва → Париж</b>\n\n🎫 <b>15 000 ₽</b>\n📅 15 дек → 22 дек\n🛫 SU • 3ч 35м\n🔗 <a href=\"https://www.aviasales.com/search/MOW1512PAR2212?marker=668475&passengers=2\">Купить билет</a>"
}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

// Тесты нового endpoint /flights/search
func TestFlightSearch_ReturnsFlights(t *testing.T) {
	flightSearcher := &mockFlightSearcher{}
	h := NewHandler(flightSearcher)

	u, _ := url.Parse("/flights/search?origin=MOW&destination=PAR&depart_date=2024-12-15&return_date=2024-12-22")
	r := httptest.NewRequest(http.MethodGet, u.String(), nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status: %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("json: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("expected success: true")
	}

	flights := response["flights"].([]interface{})
	if len(flights) != 2 {
		t.Errorf("expected 2 flights, got %d", len(flights))
	}

	count := int(response["count"].(float64))
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestFlightSearch_MissingParams_ReturnsBadRequest(t *testing.T) {
	flightSearcher := &mockFlightSearcher{}
	h := NewHandler(flightSearcher)

	// Запрос без обязательных параметров
	r := httptest.NewRequest(http.MethodGet, "/flights/search", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("json: %v", err)
	}

	if response["error"] == nil {
		t.Error("expected error message")
	}
}

// Тесты endpoint /flights/message
func TestFlightMessage_ReturnsFormattedMessage(t *testing.T) {
	flightSearcher := &mockFlightSearcher{
		mockMessage: "✈️ <b>Москва → Париж</b>\n\n🎫 <b>15 000 ₽</b>",
	}
	h := NewHandler(flightSearcher)

	u, _ := url.Parse("/flights/message?origin=MOW&destination=PAR&depart_date=2024-12-15&origin_city=Москва&dest_city=Париж&passengers=2")
	r := httptest.NewRequest(http.MethodGet, u.String(), nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status: %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("json: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("expected success: true")
	}

	message := response["message"].(string)
	if message != flightSearcher.mockMessage {
		t.Errorf("expected formatted message, got %s", message)
	}

	passengers := int(response["passengers"].(float64))
	if passengers != 2 {
		t.Errorf("expected passengers 2, got %d", passengers)
	}
}

func TestFlightSearch_ErrorFromSearcher(t *testing.T) {
	flightSearcher := &mockFlightSearcher{shouldError: true}
	h := NewHandler(flightSearcher)

	u, _ := url.Parse("/flights/search?origin=MOW&destination=PAR&depart_date=2024-12-15")
	r := httptest.NewRequest(http.MethodGet, u.String(), nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != 502 {
		t.Fatalf("expected status 502, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("json: %v", err)
	}

	if response["error"] == nil {
		t.Error("expected error message")
	}
}

func TestGETSearch_NotFoundOnWrongPath(t *testing.T) {
	flightSearcher := &mockFlightSearcher{}
	h := NewHandler(flightSearcher)
	r := httptest.NewRequest(http.MethodGet, "/wrong", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	flightSearcher := &mockFlightSearcher{}
	h := NewHandler(flightSearcher)
	r := httptest.NewRequest(http.MethodPost, "/flights/search", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != 405 {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
