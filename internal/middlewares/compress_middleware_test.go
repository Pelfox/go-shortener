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
)

// Тестирует компрессию для JSON.
func TestCompressMiddleware_GzipJsonResponse(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "hello world"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	CompressMiddleware(next).ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected Content-Encoding: gzip")
	}
	if cl := rr.Header().Get("Content-Length"); cl != "" {
		t.Fatalf("expected Content-Length to be removed, got %q", cl)
	}
	if vary := rr.Header().Get("Vary"); !strings.Contains(vary, "Accept-Encoding") {
		t.Fatalf("expected Vary to include Accept-Encoding, got %q", vary)
	}

	reader, err := gzip.NewReader(rr.Body)
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
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<html><body>Hello</body></html>"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	CompressMiddleware(next).ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected Content-Encoding: gzip")
	}

	reader, err := gzip.NewReader(rr.Body)
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
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("plain text response"))
	})

	req := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	CompressMiddleware(next).ServeHTTP(rr, req)

	if enc := rr.Header().Get("Content-Encoding"); enc != "" {
		t.Fatalf("expected no Content-Encoding, got %q", enc)
	}

	if rr.Body.String() != "plain text response" {
		t.Fatalf("expected raw body, got %q", rr.Body.String())
	}
}

// Тестируем без поддержки компрессии gzip.
func TestCompressMiddleware_NoAcceptGzip(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"data": "value"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rr := httptest.NewRecorder()
	CompressMiddleware(next).ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("did not expect Content-Encoding: gzip")
	}

	var data map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode uncompressed JSON: %v", err)
	}
}

// Тестируем невалидное тело с gzip.
func TestCompressMiddleware_InvalidGzippedRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called on invalid gzip")
	})

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("not valid gzip"))
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	CompressMiddleware(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on invalid gzip, got %d", rr.Code)
	}
}
