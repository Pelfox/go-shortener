package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Тест для создания короткой ссылки
func TestHandleCreationRequest(t *testing.T) {
	server := NewServer("localhost:8080", "")

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

// Тест для случая, когда короткая ссылка найдена в хранилище
func TestHandleShortRequest(t *testing.T) {
	server := NewServer("localhost:8080", "")

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

// Тест для случая, когда короткая ссылка не найдена в хранилище
func TestHandleShortRequestUnknown(t *testing.T) {
	server := NewServer("localhost:8080", "")

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	result := recorder.Result()
	defer result.Body.Close()
	if result.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, result.StatusCode)
	}
}
