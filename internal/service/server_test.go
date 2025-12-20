package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Pelfox/go-shortener/internal"
	"github.com/Pelfox/go-shortener/pkg/schemas"
	"github.com/rs/zerolog"
)

var testServerConfig = internal.AppConfig{
	Host:      "localhost:8080",
	URLPrefix: "",
	FilePath:  "urls.json",
}
var serverLogger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().
	Timestamp().
	Str("component", "server").
	Logger()
var middlewareLogger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().
	Timestamp().
	Str("component", "logger-middleware").
	Logger()

// Тест для создания короткой ссылки.
func TestHandleCreationRequest(t *testing.T) {
	server := NewServer(&testServerConfig, serverLogger, middlewareLogger)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://google.com"))
	req.Header.Set("Content-Type", "text/plain")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	result := recorder.Result()
	defer result.Body.Close()
	if result.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, result.StatusCode)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "http://localhost:8080/") {
		t.Fatalf("expected returned short URL, got %q", body)
	}
}

// Тест для случая с неверным Content-Type.
func TestHandleCreationRequest_InvalidContentType(t *testing.T) {
	server := NewServer(&testServerConfig, serverLogger, middlewareLogger)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://google.com"))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

// Тест для случая, когда короткая ссылка найдена в хранилище
func TestHandleShortRequest(t *testing.T) {
	server := NewServer(&testServerConfig, serverLogger, middlewareLogger)

	// добавляем фейковую короткую ссылку (mock)
	server.storage["test1234"] = "https://google.com"

	req := httptest.NewRequest(http.MethodGet, "/test1234", nil)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	result := recorder.Result()
	defer result.Body.Close()
	if result.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, result.StatusCode)
	}

	if result.Header.Get("Location") != "https://google.com" {
		t.Fatalf("expected redirect to %q, got %q", "https://google.com", result.Header.Get("Location"))
	}
}

// Тест для случая с пустым телом запроса.
func TestHandleCreationRequest_EmptyBody(t *testing.T) {
	server := NewServer(&testServerConfig, serverLogger, middlewareLogger)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("   "))
	req.Header.Set("Content-Type", "text/plain")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
}

// Тест для случая, когда короткая ссылка не найдена в хранилище.
func TestHandleShortRequestUnknown(t *testing.T) {
	server := NewServer(&testServerConfig, serverLogger, middlewareLogger)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	result := recorder.Result()
	defer result.Body.Close()
	if result.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, result.StatusCode)
	}
}

// Тестирует API создания короткой ссылки через JSON.
func TestHandleShortenRequest(t *testing.T) {
	server := NewServer(&testServerConfig, serverLogger, middlewareLogger)

	body := `{"url":"https://google.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, result.StatusCode)
	}

	if ct := result.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	respBody, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var response schemas.ShortLinkResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	if !strings.Contains(response.Result, "http://localhost:8080/") {
		t.Fatalf("expected short URL, got %q", response.Result)
	}
}

// Тестирует API создания короткой ссылки с неверным Content-Type.
func TestHandleShortenRequest_InvalidContentType(t *testing.T) {
	server := NewServer(&testServerConfig, serverLogger, middlewareLogger)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"https://google.com"}`),
	)
	req.Header.Set("Content-Type", "text/plain")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

// Тестирует API создания короткой ссылки с невалидным JSON-телом.
func TestHandleShortenRequest_InvalidJSON(t *testing.T) {
	server := NewServer(&testServerConfig, serverLogger, middlewareLogger)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

// Тестирует API создания короткой ссылки с пустым URL.
func TestHandleShortenRequest_EmptyURL(t *testing.T) {
	server := NewServer(&testServerConfig, serverLogger, middlewareLogger)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"   "}`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
}
