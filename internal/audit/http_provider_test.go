package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Тестирует успешную отправку события через HTTPProvider.
func TestHTTPProvider_Send_Success(t *testing.T) {
	var (
		gotMethod      string
		gotContentType string
		gotAction      Action
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")

		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&gotAction); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewHTTPProvider(server.URL)
	userID := "user_1"

	err := provider.Send(
		context.Background(),
		AuditActionTypeShorten,
		&userID,
		"https://example.com",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %q, got %q", http.MethodPost, gotMethod)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", gotContentType)
	}
	if gotAction.ActionType != AuditActionTypeShorten {
		t.Fatalf("expected action type %q, got %q", AuditActionTypeShorten, gotAction.ActionType)
	}
	if gotAction.UserID == nil || *gotAction.UserID != userID {
		t.Fatalf("unexpected userID in payload: %+v", gotAction.UserID)
	}
	if gotAction.URL != "https://example.com" {
		t.Fatalf("expected URL %q, got %q", "https://example.com", gotAction.URL)
	}
	if gotAction.Timestamp <= 0 {
		t.Fatalf("expected positive timestamp, got %d", gotAction.Timestamp)
	}
}

// Тестирует ошибку при недоступном endpoint.
func TestHTTPProvider_Send_RequestError(t *testing.T) {
	provider := NewHTTPProvider("http://127.0.0.1:1")

	err := provider.Send(
		context.Background(),
		AuditActionTypeFollow,
		nil,
		"https://example.com",
	)
	if err == nil {
		t.Fatal("expected error for unreachable endpoint, got nil")
	}

	if !strings.Contains(err.Error(), "failed to complete the request") {
		t.Fatalf("expected wrapped request error, got %v", err)
	}
}
