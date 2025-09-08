package httpiface

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type logEntry struct {
	level string
	event string
	data  map[string]interface{}
}

type testLogger struct{ entries []logEntry }

func (l *testLogger) Info(event string, data map[string]interface{}) {
	l.entries = append(l.entries, logEntry{level: "info", event: event, data: data})
}
func (l *testLogger) Error(event string, data map[string]interface{}) {
	l.entries = append(l.entries, logEntry{level: "error", event: event, data: data})
}

// minimal interface expected by handler implementation
// reuse loggerInterface from handlers.go

func TestFlightSearch_LogsBadRequest(t *testing.T) {
	lg := &testLogger{}
	h := NewHandlerWithLogger(&mockFlightSearcher{}, lg)

	r := httptest.NewRequest(http.MethodGet, "/flights/search", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d", w.Code)
	}

	found := false
	for _, e := range lg.entries {
		if e.level == "error" && e.event == "http_request" {
			if v, ok := e.data["status"]; !ok || v.(int) != 400 {
				t.Fatalf("expected status 400 in log")
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error log for bad request")
	}
}

func TestFlightSearch_LogsSuccessAndDuration(t *testing.T) {
	lg := &testLogger{}
	h := NewHandlerWithLogger(&mockFlightSearcher{}, lg)

	u, _ := url.Parse("/flights/search?origin=MOW&destination=PAR&depart_date=2024-12-15")
	r := httptest.NewRequest(http.MethodGet, u.String(), nil)
	w := httptest.NewRecorder()
	start := time.Now()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}

	okFound := false
	for _, e := range lg.entries {
		if e.level == "info" && e.event == "http_request" {
			if v, ok := e.data["success"]; !ok || v.(bool) != true {
				t.Fatalf("expected success=true in log")
			}
			if v, ok := e.data["count"]; !ok || int(v.(int)) != 2 {
				t.Fatalf("expected count=2 in log")
			}
			if v, ok := e.data["duration_ms"]; !ok || v.(int64) <= 0 {
				t.Fatalf("expected positive duration_ms")
			}
			if time.Since(start) < 0 {
				t.Fatalf("time monotonic check")
			}
			okFound = true
			break
		}
	}
	if !okFound {
		t.Fatalf("expected info log for success request")
	}

	var js map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &js); err != nil {
		t.Fatalf("json: %v", err)
	}
}

func TestFlightSearch_LogsUpstreamError(t *testing.T) {
	lg := &testLogger{}
	h := NewHandlerWithLogger(&mockFlightSearcher{shouldError: true}, lg)

	u, _ := url.Parse("/flights/search?origin=MOW&destination=PAR&depart_date=2024-12-15")
	r := httptest.NewRequest(http.MethodGet, u.String(), nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status: %d", w.Code)
	}

	found := false
	for _, e := range lg.entries {
		if e.level == "error" && e.event == "http_request" {
			if v, ok := e.data["status"]; !ok || v.(int) != 502 {
				t.Fatalf("expected status 502 in log")
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error log for upstream error")
	}
}
