package services

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/Pelfox/go-shortener/internal/audit"
	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/rs/zerolog"
)

func prepareShortenerService(t *testing.T) *ShortenerService {
	t.Helper()
	storageInstance := storage.NewInMemoryStorage(
		filepath.Join(t.TempDir(), "urls.json"),
	)

	return NewShortenerService(
		context.Background(),
		"http://localhost",
		storageInstance,
		zerolog.New(io.Discard),
		[]audit.Provider{},
	)
}

func prepareShortenerServiceContext(t *testing.T) context.Context {
	t.Helper()
	return context.WithValue(t.Context(), pkg.ContextUserIDKey, "user_123")
}

// Тест для случая, когда передан пустой shortID.
func TestShortenerService_GetDestination_EmptyShortID(t *testing.T) {
	service := prepareShortenerService(t)
	ctx := prepareShortenerServiceContext(t)

	_, err := service.GetDestination(ctx, "   ")
	if !errors.Is(err, ErrShortIDEmpty) {
		t.Fatalf("expected ErrShortIDEmpty, got %v", err)
	}
}

// Тест для случая, когда shortID не существует в хранилище.
func TestShortenerService_GetDestination_NotFound(t *testing.T) {
	service := prepareShortenerService(t)
	ctx := prepareShortenerServiceContext(t)

	_, err := service.GetDestination(ctx, "unknown")
	if !errors.Is(err, ErrDestinationNotFound) {
		t.Fatalf("expected ErrDestinationNotFound, got %v", err)
	}
}

// Тест успешного получения destination по shortID.
func TestShortenerService_GetDestination_Success(t *testing.T) {
	service := prepareShortenerService(t)
	ctx := prepareShortenerServiceContext(t)

	shortURL, _, err := service.CreateShortLink(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("failed to create short link: %v", err)
	}

	shortID := shortURL[len("http://localhost/"):]
	destination, err := service.GetDestination(ctx, shortID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if destination != "https://example.com" {
		t.Fatalf("expected destination https://example.com, got %q", destination)
	}
}

// Тест для случая, когда destination пустой.
func TestShortenerService_CreateShortLink_EmptyDestination(t *testing.T) {
	service := prepareShortenerService(t)
	ctx := prepareShortenerServiceContext(t)

	_, _, err := service.CreateShortLink(ctx, "   ")
	if !errors.Is(err, ErrDestinationEmpty) {
		t.Fatalf("expected ErrDestinationEmpty, got %v", err)
	}
}

// Тест создания новой короткой ссылки.
func TestShortenerService_CreateShortLink_New(t *testing.T) {
	service := prepareShortenerService(t)
	ctx := prepareShortenerServiceContext(t)

	shortURL, existed, err := service.CreateShortLink(
		ctx,
		"https://example.com",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if existed {
		t.Fatal("expected existed=false for new link")
	}

	if shortURL == "" {
		t.Fatal("expected non-empty shortURL")
	}
}

// Тест повторного сокращения уже существующей ссылки.
func TestShortenerService_CreateShortLink_AlreadyExists(t *testing.T) {
	service := prepareShortenerService(t)
	ctx := prepareShortenerServiceContext(t)

	firstURL, existed, err := service.CreateShortLink(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if existed {
		t.Fatal("first call must not mark link as existing")
	}

	secondURL, existed, err := service.CreateShortLink(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !existed {
		t.Fatal("expected existed=true for existing link")
	}

	if firstURL != secondURL {
		t.Fatalf("expected same shortURL, got %q and %q", firstURL, secondURL)
	}
}

// Тест получения списка ссылок, когда пользователь ещё ничего не создавал.
func TestShortenerService_GetUserLinks_Empty(t *testing.T) {
	service := prepareShortenerService(t)
	ctx := prepareShortenerServiceContext(t)

	links, err := service.GetUserLinks(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(links) != 0 {
		t.Fatalf("expected 0 links, got %d", len(links))
	}
}

// Тест получения списка всех ссылок пользователя.
func TestShortenerService_GetUserLinks_Success(t *testing.T) {
	service := prepareShortenerService(t)
	ctx := prepareShortenerServiceContext(t)

	_, _, err := service.CreateShortLink(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _, err = service.CreateShortLink(ctx, "https://google.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	links, err := service.GetUserLinks(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}

	for _, link := range links {
		if link.ShortURL == "" || link.OriginalURL == "" {
			t.Fatal("expected both ShortURL and OriginalURL to be non-empty")
		}
	}
}
