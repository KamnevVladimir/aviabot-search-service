package aviasales

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Тест поиска дешевых билетов за месяц
func TestClient_SearchCheap_MonthSearch(t *testing.T) {
	baseURL := "https://api.travelpayouts.com"
	captured := make(chan *http.Request, 1)

	// Мокаем ответ API в правильном формате Travelpayouts
	mockResponse := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"PAR": map[string]interface{}{
				"0": map[string]interface{}{
					"price":         15000,
					"airline":       "SU",
					"flight_number": 123,
					"departure_at":  "2024-12-15T10:30:00.000Z",
					"return_at":     "2024-12-22T15:45:00.000Z",
					"expires_at":    "2024-11-15T12:00:00.000Z",
					"origin":        "MOW",
					"destination":   "PAR",
					"gate":          "aviasales",
					"actual":        true,
					"distance":      2500,
					"duration":      215,
				},
				"1": map[string]interface{}{
					"price":         17500,
					"airline":       "AF",
					"flight_number": 456,
					"departure_at":  "2024-12-20T08:15:00.000Z",
					"return_at":     "2024-12-27T19:30:00.000Z",
					"expires_at":    "2024-11-15T12:00:00.000Z",
					"origin":        "MOW",
					"destination":   "PAR",
					"gate":          "aviasales",
					"actual":        true,
					"distance":      2500,
					"duration":      220,
				},
			},
		},
		"currency": "rub",
	}

	bodyBytes, _ := json.Marshal(mockResponse)

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		captured <- r
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(string(bodyBytes))),
			Header:     make(http.Header),
		}, nil
	})
	client := &http.Client{Transport: rt}

	c := NewClient(baseURL, "TEST_TOKEN", "668475", WithHTTPClient(client))
	params := SearchParams{
		Origin:      "MOW",
		Destination: "PAR",
		DepartDate:  "2024-12",
		Currency:    "rub",
		Limit:       3,
	}

	flights, err := c.SearchCheap(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Проверяем запрос
	req := <-captured
	if req.URL.Path != "/v1/prices/cheap" {
		t.Errorf("expected path /v1/prices/cheap, got %s", req.URL.Path)
	}

	// Проверяем параметры запроса
	query := req.URL.Query()
	if query.Get("origin") != "MOW" {
		t.Errorf("expected origin MOW, got %s", query.Get("origin"))
	}
	if query.Get("destination") != "PAR" {
		t.Errorf("expected destination PAR, got %s", query.Get("destination"))
	}
	if query.Get("depart_date") != "2024-12" {
		t.Errorf("expected depart_date 2024-12, got %s", query.Get("depart_date"))
	}
	if query.Get("token") != "TEST_TOKEN" {
		t.Errorf("expected token TEST_TOKEN, got %s", query.Get("token"))
	}
	if query.Get("marker") != "668475" {
		t.Errorf("expected marker 668475, got %s", query.Get("marker"))
	}

	// Проверяем результат
	if len(flights) == 0 {
		t.Fatal("expected flights, got empty slice")
	}

	if flights[0].Price != 15000 {
		t.Errorf("expected price 15000, got %d", flights[0].Price)
	}
	if flights[0].Origin != "MOW" {
		t.Errorf("expected origin MOW, got %s", flights[0].Origin)
	}
	if flights[0].Destination != "PAR" {
		t.Errorf("expected destination PAR, got %s", flights[0].Destination)
	}
}

// Тест поиска по точной дате
func TestClient_SearchCheap_ExactDateSearch(t *testing.T) {
	baseURL := "https://api.travelpayouts.com"
	captured := make(chan *http.Request, 1)

	mockResponse := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"PAR": map[string]interface{}{
				"0": map[string]interface{}{
					"price":        18000,
					"airline":      "SU",
					"departure_at": "2024-12-15T10:30:00.000Z",
					"return_at":    "2024-12-22T15:45:00.000Z",
					"origin":       "MOW",
					"destination":  "PAR",
					"gate":         "aviasales",
				},
			},
		},
		"currency": "rub",
	}

	bodyBytes, _ := json.Marshal(mockResponse)

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		captured <- r
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(string(bodyBytes))),
			Header:     make(http.Header),
		}, nil
	})
	client := &http.Client{Transport: rt}

	c := NewClient(baseURL, "TEST_TOKEN", "668475", WithHTTPClient(client))
	params := SearchParams{
		Origin:      "MOW",
		Destination: "PAR",
		DepartDate:  "2024-12-15",
		ReturnDate:  "2024-12-22",
		Currency:    "rub",
	}

	flights, err := c.SearchCheap(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Проверяем запрос
	req := <-captured
	query := req.URL.Query()
	if query.Get("depart_date") != "2024-12-15" {
		t.Errorf("expected depart_date 2024-12-15, got %s", query.Get("depart_date"))
	}
	if query.Get("return_date") != "2024-12-22" {
		t.Errorf("expected return_date 2024-12-22, got %s", query.Get("return_date"))
	}

	// Проверяем результат
	if len(flights) == 0 {
		t.Fatal("expected flights, got empty slice")
	}
	if flights[0].Price != 18000 {
		t.Errorf("expected price 18000, got %d", flights[0].Price)
	}
}

// Тест генерации партнерских ссылок
func TestClient_GeneratePartnerLink(t *testing.T) {
	c := NewClient("https://api.travelpayouts.com", "TEST_TOKEN", "668475")

	flight := Flight{
		Origin:      "MOW",
		Destination: "PAR",
		DepartDate:  time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC),
		ReturnDate:  time.Date(2024, 12, 22, 15, 45, 0, 0, time.UTC),
		Price:       15000,
		Airline:     "SU",
	}

	link := c.GeneratePartnerLink(flight, 2)

	expectedPrefix := "https://www.aviasales.com/search/MOW1512PAR2212"
	if !strings.HasPrefix(link, expectedPrefix) {
		t.Errorf("expected link to start with %s, got %s", expectedPrefix, link)
	}

	if !strings.Contains(link, "marker=668475") {
		t.Errorf("expected link to contain marker=668475, got %s", link)
	}

	if !strings.Contains(link, "passengers=2") {
		t.Errorf("expected link to contain passengers=2, got %s", link)
	}
}

// Тест обработки ошибок API
func TestClient_SearchCheap_APIError(t *testing.T) {
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 401,
			Body:       io.NopCloser(strings.NewReader(`{"success": false, "error": "Invalid token"}`)),
			Header:     make(http.Header),
		}, nil
	})
	client := &http.Client{Transport: rt}

	c := NewClient("https://api.travelpayouts.com", "INVALID_TOKEN", "668475", WithHTTPClient(client))
	params := SearchParams{
		Origin:      "MOW",
		Destination: "PAR",
		DepartDate:  "2024-12",
		Currency:    "rub",
	}

	_, err := c.SearchCheap(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

// Тест обработки сетевых ошибок
func TestClient_SearchCheap_NetworkError(t *testing.T) {
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("network error")
	})
	client := &http.Client{Transport: rt}

	c := NewClient("https://api.travelpayouts.com", "TEST_TOKEN", "668475", WithHTTPClient(client))
	params := SearchParams{
		Origin:      "MOW",
		Destination: "PAR",
		DepartDate:  "2024-12",
		Currency:    "rub",
	}

	_, err := c.SearchCheap(context.Background(), params)
	if err == nil {
		t.Fatal("expected network error")
	}
}

// Тест форматирования сообщений с билетами
func TestClient_FormatFlightMessage(t *testing.T) {
	c := NewClient("https://api.travelpayouts.com", "TEST_TOKEN", "668475")

	flights := []Flight{
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
	}

	message := c.FormatFlightMessage("Москва", "Париж", flights, 2)

	// Проверяем что сообщение содержит основную информацию
	if !strings.Contains(message, "Москва → Париж") {
		t.Error("message should contain route")
	}
	if !strings.Contains(message, "15 000 ₽") {
		t.Error("message should contain first flight price")
	}
	if !strings.Contains(message, "17 500 ₽") {
		t.Error("message should contain second flight price")
	}
	if !strings.Contains(message, "15 дек") {
		t.Error("message should contain departure date")
	}
	if !strings.Contains(message, "22 дек") {
		t.Error("message should contain return date")
	}
	if !strings.Contains(message, "SU") {
		t.Error("message should contain airline")
	}
	if !strings.Contains(message, "3ч 35м") {
		t.Error("message should contain duration")
	}
	if !strings.Contains(message, "Купить билет") {
		t.Error("message should contain purchase links")
	}
}
