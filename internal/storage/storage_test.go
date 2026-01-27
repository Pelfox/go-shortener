package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Pelfox/go-shortener/pkg"
)

func prepareTestStorage(t *testing.T) (*InMemoryStorage, context.Context) {
	t.Helper()
	ctx := context.WithValue(t.Context(), pkg.ContextUserIDKey, "user_123")
	storage := NewInMemoryStorage(filepath.Join(t.TempDir(), "storage.json"))
	return storage, ctx
}

// Тестирует сохранение и получение значений в InMemoryStorage.
func TestInMemoryStorage_StoreAndGet(t *testing.T) {
	storage, ctx := prepareTestStorage(t)

	err := storage.Store(ctx, "abc123", "https://google.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value, err := storage.Get(ctx, "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "https://google.com" {
		t.Fatalf("expected %q, got %q", "https://google.com", value)
	}
}

// Тестирует поиск короткой ссылки по исходному URL.
func TestInMemoryStorage_GetByDestination(t *testing.T) {
	storage, ctx := prepareTestStorage(t)

	if err := storage.Store(ctx, "abc123", "https://example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	id, err := storage.GetByDestination(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id != "abc123" {
		t.Fatalf("expected %q, got %q", "abc123", id)
	}

	if _, err := storage.GetByDestination(ctx, "https://missing.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Тестирует обработку коллизий при сохранении в InMemoryStorage.
func TestInMemoryStorage_StoreCollision(t *testing.T) {
	storage, ctx := prepareTestStorage(t)

	if err := storage.Store(ctx, "id", "url1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := storage.Store(ctx, "id", "url2")
	if !errors.Is(err, ErrIDCollision) {
		t.Fatalf("expected ErrIDCollision, got %v", err)
	}
}

// Тестирует получение несуществующего ключа из InMemoryStorage.
func TestInMemoryStorage_GetNotFound(t *testing.T) {
	storage, ctx := prepareTestStorage(t)

	_, err := storage.Get(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Тестирует сохранение и загрузку данных из файла в InMemoryStorage.
func TestInMemoryStorage_SaveAndLoad(t *testing.T) {
	storage, ctx := prepareTestStorage(t)

	if err := storage.Store(ctx, "id1", "https://example.com"); err != nil {
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

	value, err := newStorage.Get(ctx, "id1")
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
