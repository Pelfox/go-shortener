package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Тестирует, что Send добавляет событие в буфер без ошибок.
func TestFileProvider_Send(t *testing.T) {
	provider := NewFileProvider(filepath.Join(t.TempDir(), "audit.jsonl"))

	userID := "user_123"
	err := provider.Send(
		context.Background(),
		AuditActionTypeShorten,
		&userID,
		"https://example.com",
	)
	if err != nil {
		t.Fatalf("expected no error from Send, got %v", err)
	}

	if len(provider.entries) != 1 {
		t.Fatalf("expected 1 buffered entry, got %d", len(provider.entries))
	}

	entry := provider.entries[0]
	if entry.ActionType != AuditActionTypeShorten {
		t.Fatalf("expected action type %q, got %q", AuditActionTypeShorten, entry.ActionType)
	}
	if entry.UserID == nil || *entry.UserID != userID {
		t.Fatalf("expected userID %q, got %+v", userID, entry.UserID)
	}
	if entry.URL != "https://example.com" {
		t.Fatalf("expected URL %q, got %q", "https://example.com", entry.URL)
	}
	if entry.Timestamp <= 0 {
		t.Fatalf("expected positive timestamp, got %d", entry.Timestamp)
	}
}

// Тестирует, что Close при пустом буфере возвращает nil и не падает.
func TestFileProvider_Close_EmptyEntries(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "audit.jsonl")
	provider := NewFileProvider(filePath)

	if err := provider.Close(); err != nil {
		t.Fatalf("expected no error on empty Close, got %v", err)
	}
}

// Тестирует, что Close записывает события в JSONL и очищает буфер.
func TestFileProvider_Close_WritesEntries(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "audit.jsonl")
	provider := NewFileProvider(filePath)

	userID := "user_1"
	if err := provider.Send(context.Background(), AuditActionTypeShorten, &userID, "https://example.com"); err != nil {
		t.Fatalf("unexpected Send error: %v", err)
	}
	if err := provider.Send(context.Background(), AuditActionTypeFollow, nil, "https://google.com"); err != nil {
		t.Fatalf("unexpected Send error: %v", err)
	}

	if err := provider.Close(); err != nil {
		t.Fatalf("unexpected Close error: %v", err)
	}

	if provider.entries != nil {
		t.Fatalf("expected entries buffer to be reset to nil after Close")
	}

	fileContentScanner := bufio.NewScanner(strings.NewReader(mustReadFile(t, filePath)))
	var lines []string
	for fileContentScanner.Scan() {
		lines = append(lines, fileContentScanner.Text())
	}
	if err := fileContentScanner.Err(); err != nil {
		t.Fatalf("failed to scan written file: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d", len(lines))
	}

	var first Action
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("failed to unmarshal first line: %v", err)
	}
	if first.ActionType != AuditActionTypeShorten {
		t.Fatalf("expected first action %q, got %q", AuditActionTypeShorten, first.ActionType)
	}
	if first.UserID == nil || *first.UserID != "user_1" {
		t.Fatalf("unexpected first userID: %+v", first.UserID)
	}
	if first.URL != "https://example.com" {
		t.Fatalf("unexpected first URL: %q", first.URL)
	}

	var second Action
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("failed to unmarshal second line: %v", err)
	}
	if second.ActionType != AuditActionTypeFollow {
		t.Fatalf("expected second action %q, got %q", AuditActionTypeFollow, second.ActionType)
	}
	if second.UserID != nil {
		t.Fatalf("expected second userID to be nil, got %+v", second.UserID)
	}
	if second.URL != "https://google.com" {
		t.Fatalf("unexpected second URL: %q", second.URL)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %q: %v", path, err)
	}
	return string(data)
}
