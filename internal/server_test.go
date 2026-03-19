package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/Pelfox/go-shortener/pkg/schemas"
	"github.com/rs/zerolog"
)

func prepareServer(t *testing.T) *Server {
	t.Helper()
	tempDir := t.TempDir()

	storageInstance := storage.NewInMemoryStorage(filepath.Join(tempDir, "urls.json"))
	auditFile := filepath.Join(tempDir, "audit.json")

	config := AppConfig{
		Addr:      "localhost:8080",
		BaseURL:   "http://localhost:8080/test",
		Secret:    []byte("very-strong-secret"),
		AuditFile: auditFile,
	}
	return NewServer(
		&config,
		zerolog.New(io.Discard),
		storageInstance,
		nil,
	)
}

// Тест успешного создания короткой ссылки через text/plain API.
func TestServer_PlainCreate_OK(t *testing.T) {
	server := prepareServer(t)

	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("https://example.com"),
	)
	request.Header.Set("Content-Type", "text/plain")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}

	if !strings.HasPrefix(recorder.Body.String(), "http://localhost:8080/test") {
		t.Fatalf("unexpected response body: %q", recorder.Body.String())
	}
}

// Тест ошибки при неверном Content-Type для text/plain API.
func TestServer_PlainCreate_InvalidContentType(t *testing.T) {
	server := prepareServer(t)

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

// Тест успешного создания короткой ссылки через JSON API.
func TestServer_APIShorten_OK(t *testing.T) {
	server := prepareServer(t)

	body := `{"url":"https://example.com"}`
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}

	var response schemas.ShortLinkResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	if !strings.HasPrefix(response.Result, "http://localhost:8080/test") {
		t.Fatalf("unexpected short URL: %q", response.Result)
	}
}

// Тест конфликта при повторном сокращении той же ссылки.
func TestServer_APIShorten_Conflict(t *testing.T) {
	server := prepareServer(t)

	// первый запрос
	request1 := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"https://example.com"}`),
	)
	request1.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(httptest.NewRecorder(), request1)

	// повторный
	request2 := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"https://example.com"}`),
	)
	request2.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request2)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
}

// Тест batch-сокращения ссылок с конфликтом.
func TestServer_APIBatch_Conflict(t *testing.T) {
	server := prepareServer(t)

	payload := []schemas.BatchedLinkRequest{
		{CorrelationID: "1", OriginalURL: "https://example.com"},
		{CorrelationID: "2", OriginalURL: "https://example.com"},
	}
	body, _ := json.Marshal(payload)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten/batch",
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
}

// Тест получения пустого списка ссылок пользователя.
func TestServer_GetUserLinks_NoContent(t *testing.T) {
	server := prepareServer(t)

	request := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	recorder := httptest.NewRecorder()

	server.router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
}

// Тест успешной переадресации по короткой ссылке.
func TestServer_Redirect_OK(t *testing.T) {
	server := prepareServer(t)

	// создаём ссылку
	reqCreate := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("https://example.com"),
	)
	reqCreate.Header.Set("Content-Type", "text/plain")
	recCreate := httptest.NewRecorder()
	server.router.ServeHTTP(recCreate, reqCreate)

	shortURL := strings.TrimSpace(recCreate.Body.String())
	shortID := strings.TrimPrefix(shortURL, "http://localhost:8080/test/")

	request := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	recorder := httptest.NewRecorder()

	server.router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", recorder.Code)
	}

	if recorder.Header().Get("Location") != "https://example.com" {
		t.Fatalf("unexpected redirect location")
	}
}

// Тест health-check без подключённой базы данных.
func TestServer_Health_NoDB(t *testing.T) {
	server := prepareServer(t)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
