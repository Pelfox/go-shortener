package services

import (
	"context"
	"testing"

	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/rs/zerolog"
)

func BenchmarkCreateShortLink(b *testing.B) {
	ctx := context.WithValue(context.Background(), pkg.ContextUserIDKey, "test-user-123")
	logger := zerolog.Nop()
	store := storage.NewInMemoryStorage("")
	service := NewShortenerService(ctx, "http://localhost:8080", store, logger, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _, _ = service.CreateShortLink(ctx, "https://example.com/test-benchmark")
	}
}

func BenchmarkGetDestination(b *testing.B) {
	ctx := context.WithValue(context.Background(), pkg.ContextUserIDKey, "test-user-123")
	logger := zerolog.Nop()
	store := storage.NewInMemoryStorage("")
	service := NewShortenerService(ctx, "http://localhost:8080", store, logger, nil)

	shortID := "short123"
	_ = store.Store(ctx, shortID, "https://example.com/test-benchmark")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = service.GetDestination(ctx, shortID)
	}
}

func BenchmarkGetUserLinks(b *testing.B) {
	ctx := context.WithValue(context.Background(), pkg.ContextUserIDKey, "test-user-123")
	logger := zerolog.Nop()
	store := storage.NewInMemoryStorage("")
	service := NewShortenerService(ctx, "http://localhost:8080", store, logger, nil)

	_ = store.Store(ctx, "short123", "https://example.com/test-1")
	_ = store.Store(ctx, "short124", "https://example.com/test-2")
	_ = store.Store(ctx, "short125", "https://example.com/test-3")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = service.GetUserLinks(ctx)
	}
}
