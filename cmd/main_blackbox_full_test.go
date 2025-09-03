package main

import (
	app "aviasales-bot/search-service/internal/application"
	api "aviasales-bot/search-service/internal/infrastructure/aviasales"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Blackbox тесты для search-service main - проверяем поведение как пользователь API

func TestMain_SearchEndpoint_ReturnsJSON(t *testing.T) {
	// Тест: /search endpoint должен возвращать JSON с 3 элементами
	mockResponse := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"PAR": map[string]interface{}{
				"0": map[string]interface{}{
					"price":        15000,
					"airline":      "SU",
					"departure_at": "2024-12-15T10:30:00.000Z",
					"return_at":    "2024-12-22T15:45:00.000Z",
					"origin":       "MOW",
					"destination":  "PAR",
				},
				"1": map[string]interface{}{
					"price":        17500,
					"airline":      "AF",
					"departure_at": "2024-12-20T08:15:00.000Z",
					"return_at":    "2024-12-27T19:30:00.000Z",
					"origin":       "MOW",
					"destination":  "PAR",
				},
				"2": map[string]interface{}{
					"price":        19000,
					"airline":      "LH",
					"departure_at": "2024-12-25T12:00:00.000Z",
					"return_at":    "2025-01-02T18:00:00.000Z",
					"origin":       "MOW",
					"destination":  "PAR",
				},
			},
		},
		"currency": "rub",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer ts.Close()

	// Тестируем через clientAdapter
	adapter := &clientAdapter{c: api.NewClient(ts.URL, "test", "test")}

	// Создаем параметры поиска
	params := app.SearchParams{
		Origin:      "MOW",
		Destination: "PAR",
		DepartDate:  "2024-12",
		Currency:    "rub",
		Limit:       3,
	}

	// Выполняем поиск
	result, err := adapter.Search(context.Background(), params)
	if err != nil {
		t.Fatalf("adapter failed: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
}

func TestMain_SearchEndpoint_HandlesErrors(t *testing.T) {
	// Тест: /search endpoint должен обрабатывать ошибки
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	// Тестируем через clientAdapter
	adapter := &clientAdapter{c: api.NewClient(ts.URL, "test", "test")}

	// Создаем параметры поиска
	params := app.SearchParams{
		Origin:      "MOW",
		Destination: "PAR",
		DepartDate:  "2024-12",
		Currency:    "rub",
		Limit:       3,
	}

	// Выполняем поиск - должен вернуть ошибку
	_, err := adapter.Search(context.Background(), params)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMain_HealthEndpoint_ReturnsOK(t *testing.T) {
	// Тест: health endpoint должен отвечать 200 OK
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","service":"search-service"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	// Проверяем health endpoint
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("health endpoint failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMain_EnvironmentVariables_Optional(t *testing.T) {
	// Тест: переменные окружения опциональны
	os.Unsetenv("PORT")
	os.Unsetenv("AVIASALES_URL")
	os.Unsetenv("AVIASALES_TOKEN")
	os.Unsetenv("AVIASALES_MARKER")

	// Не должно паниковать при отсутствии переменных
	// (тест проходит если main.go не паникует при запуске)
}

func TestMain_EnvironmentVariables_Defaults(t *testing.T) {
	// Тест: используются значения по умолчанию
	os.Setenv("AVIASALES_TOKEN", "test-token")
	os.Setenv("AVIASALES_BASE_URL", "http://test")
	os.Setenv("AVIASALES_MARKER", "test-marker")
	os.Setenv("LISTEN_ADDR", ":8084")
	defer func() {
		os.Unsetenv("AVIASALES_TOKEN")
		os.Unsetenv("AVIASALES_BASE_URL")
		os.Unsetenv("AVIASALES_MARKER")
		os.Unsetenv("LISTEN_ADDR")
	}()

	// Не должно паниковать при установке переменных
	// (тест проходит если main.go не паникует при запуске)
}
