package middlewares

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func prepareCompressLogger(t *testing.T) zerolog.Logger {
	t.Helper()
	return zerolog.New(io.Discard)
}

// Тестирует компрессию для JSON.
func TestCompressMiddleware_GzipJsonResponse(t *testing.T) {
	logger := prepareCompressLogger(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "hello world"})
	})

	request := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	recorder := httptest.NewRecorder()
	CompressMiddleware(logger)(next).ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected Content-Encoding: gzip")
	}
	if cl := recorder.Header().Get("Content-Length"); cl != "" {
		t.Fatalf("expected Content-Length to be removed, got %q", cl)
	}
	if vary := recorder.Header().Get("Vary"); !strings.Contains(vary, "Accept-Encoding") {
		t.Fatalf("expected Vary to include Accept-Encoding, got %q", vary)
	}

	reader, err := gzip.NewReader(recorder.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read gzipped body: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if data["message"] != "hello world" {
		t.Fatalf("expected message 'hello world', got %v", data["message"])
	}
}

// Тестирует компрессию для HTML.
func TestCompressMiddleware_GzipHtmlResponse(t *testing.T) {
	logger := prepareCompressLogger(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<html><body>Hello</body></html>"))
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	recorder := httptest.NewRecorder()
	CompressMiddleware(logger)(next).ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected Content-Encoding: gzip")
	}

	reader, err := gzip.NewReader(recorder.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	if !bytes.Contains(body, []byte("Hello")) {
		t.Fatal("expected HTML body to contain 'Hello'")
	}
}

// Тестируем отсутствие компрессии для других типов контента.
func TestCompressMiddleware_NoGzipForOtherContentTypes(t *testing.T) {
	logger := prepareCompressLogger(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("plain text response"))
	})

	request := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	recorder := httptest.NewRecorder()
	CompressMiddleware(logger)(next).ServeHTTP(recorder, request)

	if enc := recorder.Header().Get("Content-Encoding"); enc != "" {
		t.Fatalf("expected no Content-Encoding, got %q", enc)
	}

	if recorder.Body.String() != "plain text response" {
		t.Fatalf("expected raw body, got %q", recorder.Body.String())
	}
}

// Тестируем без поддержки компрессии gzip.
func TestCompressMiddleware_NoAcceptGzip(t *testing.T) {
	logger := prepareCompressLogger(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"data": "value"})
	})

	request := httptest.NewRequest(http.MethodGet, "/api", nil)
	recorder := httptest.NewRecorder()
	CompressMiddleware(logger)(next).ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("did not expect Content-Encoding: gzip")
	}

	var data map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode uncompressed JSON: %v", err)
	}
}

// Тестируем невалидное тело с gzip.
func TestCompressMiddleware_InvalidGzippedRequest(t *testing.T) {
	logger := prepareCompressLogger(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called on invalid gzip")
	})

	request := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("not valid gzip"))
	request.Header.Set("Content-Encoding", "gzip")

	recorder := httptest.NewRecorder()
	CompressMiddleware(logger)(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected '400 Bad Request' on invalid gzip, got %d", recorder.Code)
	}
}
