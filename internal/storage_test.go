package internal

import (
	"errors"
	"path/filepath"
	"testing"
)

func newTestStorage(t *testing.T) *InMemoryStorage {
	t.Helper()
	return NewInMemoryStorage(filepath.Join(t.TempDir(), "storage.json"))
}

// Тестирует сохранение и получение значений в InMemoryStorage.
func TestInMemoryStorage_StoreAndGet(t *testing.T) {
	storage := newTestStorage(t)

	err := storage.Store("abc123", "https://google.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value, err := storage.Get("abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "https://google.com" {
		t.Fatalf("expected %q, got %q", "https://google.com", value)
	}
}

// Тестирует обработку коллизий при сохранении в InMemoryStorage.
func TestInMemoryStorage_StoreCollision(t *testing.T) {
	storage := newTestStorage(t)
	if err := storage.Store("id", "url1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := storage.Store("id", "url2")
	if !errors.Is(err, ErrIDCollision) {
		t.Fatalf("expected ErrIDCollision, got %v", err)
	}
}

// Тестирует получение несуществующего ключа из InMemoryStorage.
func TestInMemoryStorage_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	_, err := storage.Get("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Тестирует сохранение и загрузку данных из файла в InMemoryStorage.
func TestInMemoryStorage_SaveAndLoad(t *testing.T) {
	storage := newTestStorage(t)
	if err := storage.Store("id1", "https://example.com"); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	if err := storage.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// создаём новый storage с тем же файлом
	newStorage := NewInMemoryStorage(storage.filePath)
	if err := newStorage.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	value, err := newStorage.Get("id1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "https://example.com" {
		t.Fatalf("expected %q, got %q", "https://example.com", value)
	}
}

// Тестирует загрузку из несуществующего файла в InMemoryStorage.
func TestInMemoryStorage_LoadNoFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "missing.json")

	storage := NewInMemoryStorage(file)
	if err := storage.Load(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
