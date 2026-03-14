package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/Pelfox/go-shortener/pkg"
)

func BenchmarkInMemoryStorage_Store(b *testing.B) {
	store := NewInMemoryStorage("")
	ctx := context.WithValue(context.Background(), pkg.ContextUserIDKey, "user123")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; b.Loop(); i++ {
		_ = store.Store(ctx, fmt.Sprintf("short%d", i), "https://example.com/test")
	}
}

func BenchmarkInMemoryStorage_Get(b *testing.B) {
	store := NewInMemoryStorage("")
	ctx := context.Background()
	shortID := "short123"
	ctxWithUser := context.WithValue(ctx, pkg.ContextUserIDKey, "user123")
	_ = store.Store(ctxWithUser, shortID, "https://example.com/test")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = store.Get(ctx, shortID)
	}
}

func BenchmarkInMemoryStorage_GetByDestination(b *testing.B) {
	store := NewInMemoryStorage("")
	ctx := context.Background()
	dest := "https://example.com/test"
	ctxWithUser := context.WithValue(ctx, pkg.ContextUserIDKey, "user123")
	_ = store.Store(ctxWithUser, "short123", dest)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = store.GetByDestination(ctx, dest)
	}
}

func BenchmarkInMemoryStorage_GetForUser(b *testing.B) {
	store := NewInMemoryStorage("")
	userID := "user123"
	ctx := context.WithValue(context.Background(), pkg.ContextUserIDKey, userID)

	for i := range 100 {
		_ = store.Store(ctx, fmt.Sprintf("short%d", i), fmt.Sprintf("https://example.com/test/%d", i))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = store.GetForUser(ctx)
	}
}

func BenchmarkInMemoryStorage_MarkDelete(b *testing.B) {
	store := NewInMemoryStorage("")
	userID := "user123"
	ctx := context.WithValue(context.Background(), pkg.ContextUserIDKey, userID)

	shortIDs := make([]string, 0, 10)
	for i := range 10 {
		id := fmt.Sprintf("short%d", i)
		_ = store.Store(ctx, id, fmt.Sprintf("https://example.com/test/%d", i))
		shortIDs = append(shortIDs, id)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = store.MarkDelete(ctx, userID, shortIDs)
	}
}
