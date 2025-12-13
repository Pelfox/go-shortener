package middlewares

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupLogger() *bytes.Buffer {
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()
	return &buf
}

// Тестирует LoggerMiddleware для 200-х запросов.
func TestLoggerMiddleware(t *testing.T) {
	buf := setupLogger()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	LoggerMiddleware(next).ServeHTTP(rr, req)

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
	buf := setupLogger()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rr := httptest.NewRecorder()
	LoggerMiddleware(next).ServeHTTP(rr, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if entry["status"] != float64(404) {
		t.Fatalf("expected status 400, got %v", entry["status"])
	}
}
