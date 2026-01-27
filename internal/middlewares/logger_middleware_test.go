package middlewares

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func prepareLogger(t *testing.T) (*bytes.Buffer, zerolog.Logger) {
	t.Helper()
	var buf bytes.Buffer
	return &buf, zerolog.New(&buf).With().Timestamp().Logger()
}

// Тестирует LoggerMiddleware для 200-х запросов.
func TestLoggerMiddleware(t *testing.T) {
	buf, logger := prepareLogger(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	LoggerMiddleware(logger)(next).ServeHTTP(recorder, request)

	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if logEntry["method"] != "GET" {
		t.Fatalf("expected method to be GET, got %v", logEntry["method"])
	}

	if logEntry["path"] != "/test" {
		t.Fatalf("expected path to be /test, got %v", logEntry["path"])
	}

	if logEntry["status"] != float64(200) {
		t.Fatalf("expected status to be 200, got %v", logEntry["status"])
	}

	if logEntry["size"] != float64(2) {
		t.Fatalf("expected size to be 2, got %v", logEntry["size"])
	}
}

// Тестирует LoggerMiddleware для 404 запросов.
func TestLoggerMiddleware_NotFound(t *testing.T) {
	buf, logger := prepareLogger(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	recorder := httptest.NewRecorder()
	LoggerMiddleware(logger)(next).ServeHTTP(recorder, request)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if entry["status"] != float64(404) {
		t.Fatalf("expected status 404, got %v", entry["status"])
	}
}
