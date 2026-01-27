package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pelfox/go-shortener/internal"
	"github.com/Pelfox/go-shortener/pkg/schemas"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

var testServerConfig = internal.AppConfig{
	Addr:        "localhost:8080",
	BaseURL:     "http://localhost:8080/",
	FilePath:    "urls.json",
	DatabaseDSN: "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
}
var serverLogger = zerolog.Nop()
var middlewareLogger = zerolog.Nop()

// Создаём отдельный экземпляр хранилища для тестов.
func createTestStorage(t *testing.T) internal.Storage {
	t.Helper()
	tempDir := t.TempDir()
	return internal.NewInMemoryStorage(filepath.Join(tempDir, testServerConfig.FilePath))
}

// Создаём отдельный экземпляр пула подключений к базе данных для тестов.
func createDatabasePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(
		t.Context(),
		testServerConfig.DatabaseDSN,
	)
	if err != nil {
		t.Fatalf("failed to create database pool: %v", err)
	}
	return pool
}

// Тест для создания короткой ссылки.
func TestHandleCreationRequest(t *testing.T) {
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

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
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://google.com"))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

// Тест для случая с уже существующей короткой ссылкой.
func TestHandleCreationRequest_Conflict(t *testing.T) {
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

	if err := server.storage.Store(context.Background(), "test1234", "https://google.com"); err != nil {
		t.Fatalf("failed to store test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://google.com"))
	req.Header.Set("Content-Type", "text/plain")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}

	body := strings.TrimSpace(recorder.Body.String())
	if body != "http://localhost:8080/test1234" {
		t.Fatalf("expected existing short URL, got %q", body)
	}
}

// Тест для случая, когда короткая ссылка найдена в хранилище
func TestHandleShortRequest(t *testing.T) {
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

	// добавляем фейковую короткую ссылку (mock)
	if err := server.storage.Store(context.Background(), "test1234", "https://google.com"); err != nil {
		t.Fatalf("failed to store test data: %v", err)
	}

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
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("   "))
	req.Header.Set("Content-Type", "text/plain")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

// Тест для случая, когда короткая ссылка не найдена в хранилище.
func TestHandleShortRequestUnknown(t *testing.T) {
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

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
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

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

// Тестирует API для уже существующей короткой ссылки через JSON.
func TestHandleShortenRequest_Conflict(t *testing.T) {
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

	if err := server.storage.Store(context.Background(), "test1234", "https://google.com"); err != nil {
		t.Fatalf("failed to store initial data: %v", err)
	}

	body := `{"url":"https://google.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, result.StatusCode)
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

	if response.Result != "http://localhost:8080/test1234" {
		t.Fatalf("expected existing short URL, got %q", response.Result)
	}
}

// Тестирует API создания короткой ссылки с неверным Content-Type.
func TestHandleShortenRequest_InvalidContentType(t *testing.T) {
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

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
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

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
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"   "}`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

// Тестирует batched API при конфликте.
func TestHandleBatchRequest_Conflict(t *testing.T) {
	server := NewServer(
		testServerConfig.Addr,
		testServerConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		createTestStorage(t),
		createDatabasePool(t),
	)

	if err := server.storage.Store(context.Background(), "existing", "https://existing.com"); err != nil {
		t.Fatalf("failed to store initial data: %v", err)
	}

	request := []schemas.BatchedLinkRequest{
		{CorrelationID: "1", OriginalURL: "https://existing.com"},
		{CorrelationID: "2", OriginalURL: "https://new.com"},
	}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, result.StatusCode)
	}

	if ct := result.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	respBody, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var response []schemas.BatchedLinkResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 results, got %d", len(response))
	}

	results := map[string]string{}
	for _, item := range response {
		results[item.CorrelationID] = item.ShortURL
	}

	if results["1"] != "http://localhost:8080/existing" {
		t.Fatalf("expected existing short URL, got %q", results["1"])
	}
	if result := results["2"]; !strings.HasPrefix(result, "http://localhost:8080/") {
		t.Fatalf("expected generated short URL, got %q", result)
	}
}
