package aviasales

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// testLogger is a lightweight mock used only in tests to verify logging calls
type testLogger struct {
	lastExternal struct {
		apiName    string
		endpoint   string
		statusCode int
		duration   time.Duration
		metadata   map[string]interface{}
		called     bool
	}
}

func (l *testLogger) ExternalAPI(apiName, endpoint string, statusCode int, duration time.Duration, metadata map[string]interface{}) error {
	l.lastExternal.apiName = apiName
	l.lastExternal.endpoint = endpoint
	l.lastExternal.statusCode = statusCode
	l.lastExternal.duration = duration
	l.lastExternal.metadata = metadata
	l.lastExternal.called = true
	return nil
}

// RoundTripper mock
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClient_LogsExternalAPI_Success(t *testing.T) {
	// mock successful API response
	mockBody, _ := json.Marshal(map[string]interface{}{
		"success":  true,
		"data":     map[string]interface{}{"PAR": map[string]interface{}{"0": map[string]interface{}{"price": 10000, "origin": "MOW", "destination": "PAR", "departure_at": "2024-12-15T10:30:00.000Z", "return_at": "2024-12-22T15:45:00.000Z"}}},
		"currency": "rub",
	})

	client := &http.Client{Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(mockBody))), Header: make(http.Header)}, nil
	})}

	c := NewClient("https://api.travelpayouts.com", "TEST", "668475", WithHTTPClient(client), WithLogger(&testLogger{}))

	_, _ = c.SearchCheap(context.Background(), SearchParams{Origin: "MOW", Destination: "PAR", DepartDate: "2024-12", Currency: "rub", Limit: 1})

	lg := c.logger.(*testLogger)
	if !lg.lastExternal.called {
		t.Fatalf("expected ExternalAPI logging to be called")
	}
	if lg.lastExternal.apiName != "travelpayouts" {
		t.Errorf("apiName: %s", lg.lastExternal.apiName)
	}
	if lg.lastExternal.endpoint != "/v1/prices/cheap" {
		t.Errorf("endpoint: %s", lg.lastExternal.endpoint)
	}
	if lg.lastExternal.statusCode != 200 {
		t.Errorf("statusCode: %d", lg.lastExternal.statusCode)
	}
}

func TestClient_LogsExternalAPI_ErrorStatus(t *testing.T) {
	client := &http.Client{Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(`{"success":false,"error":"boom"}`)), Header: make(http.Header)}, nil
	})}

	c := NewClient("https://api.travelpayouts.com", "TEST", "668475", WithHTTPClient(client), WithLogger(&testLogger{}))

	_, _ = c.SearchCheap(context.Background(), SearchParams{Origin: "MOW", Destination: "PAR", DepartDate: "2024-12"})

	lg := c.logger.(*testLogger)
	if !lg.lastExternal.called {
		t.Fatalf("expected ExternalAPI logging to be called on error status")
	}
	if lg.lastExternal.statusCode != 500 {
		t.Errorf("statusCode: %d", lg.lastExternal.statusCode)
	}
}
