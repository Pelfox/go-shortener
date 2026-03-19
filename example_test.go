package shortener_test

import (
	"context"
	"fmt"

	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/rs/zerolog"
)

// ExampleShortenerService_GetDestination демонстрирует, как можно получить
// оригинальный URL по его короткому идентификатору.
func ExampleShortenerService_GetDestination() {
	ctx := context.Background()
	logger := zerolog.Nop()
	store := storage.NewInMemoryStorage("")

	ctxWithUser := context.WithValue(ctx, pkg.ContextUserIDKey, "user-123")
	_ = store.Store(ctxWithUser, "my-short-id", "https://example.com/very/long/url")
	shortenerService := services.NewShortenerService(ctx, "http://localhost:8080", store, logger, nil)

	destination, err := shortenerService.GetDestination(ctx, "my-short-id")
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		return
	}

	fmt.Println(destination)

	// Output:
	// https://example.com/very/long/url
}

// ExampleShortenerService_CreateShortLink показывает, как создать новую
// короткую ссылку для переданного URL.
func ExampleShortenerService_CreateShortLink() {
	ctx := context.Background()
	logger := zerolog.Nop()
	store := storage.NewInMemoryStorage("")

	ctxWithUser := context.WithValue(ctx, pkg.ContextUserIDKey, "user-123")
	shortenerService := services.NewShortenerService(ctx, "http://localhost:8080", store, logger, nil)

	shortURL, conflict, err := shortenerService.CreateShortLink(ctxWithUser, "https://example.com/something/new")
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		return
	}

	fmt.Printf("Ссылка создана успешно: %v\n", err == nil)
	fmt.Printf("Был ли конфликт (ссылка уже существовала): %v\n", conflict)
	fmt.Printf("Длина сгенерированной ссылки больше нуля: %v\n", len(shortURL) > 0)

	// Output:
	// Ссылка создана успешно: true
	// Был ли конфликт (ссылка уже существовала): false
	// Длина сгенерированной ссылки больше нуля: true
}

// ExampleShortenerService_GetUserLinks демонстрирует, как получить все ссылки,
// созданные конкретным пользователем.
func ExampleShortenerService_GetUserLinks() {
	ctx := context.Background()
	logger := zerolog.Nop()
	store := storage.NewInMemoryStorage("")

	userID := "user-123"
	ctxWithUser := context.WithValue(ctx, pkg.ContextUserIDKey, userID)
	shortenerService := services.NewShortenerService(ctx, "http://localhost:8080", store, logger, nil)

	_, _, _ = shortenerService.CreateShortLink(ctxWithUser, "https://example.com/one")
	_, _, _ = shortenerService.CreateShortLink(ctxWithUser, "https://example.com/two")

	links, err := shortenerService.GetUserLinks(ctxWithUser)
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		return
	}
	fmt.Printf("Найдено ссылок: %d\n", len(links))

	// Output:
	// Найдено ссылок: 2
}

// ExampleUserService_CreateUserCookie показывает процесс генерации нового
// уникального ID пользователя и подписанного значения для Cookie.
func ExampleUserService_CreateUserCookie() {
	secret := []byte("super-secret-key-123456789")
	userService := services.NewUserService(secret)
	userID, cookieValue := userService.CreateUserCookie()

	fmt.Printf("Сгенерирован ID (длина): %d\n", len(userID))
	fmt.Printf("Сгенерирована Cookie (длина): %d\n", len(cookieValue))

	// Output:
	// Сгенерирован ID (длина): 16
	// Сгенерирована Cookie (длина): 60
}

// ExampleUserService_VerifyUserCookieValue демонстрирует валидацию
// подписанного значения Cookie и извлечение из него ID пользователя.
func ExampleUserService_VerifyUserCookieValue() {
	secret := []byte("super-secret-key-123456789")
	userService := services.NewUserService(secret)

	originalUserID, validCookie := userService.CreateUserCookie()
	extractedUserID, err := userService.VerifyUserCookieValue(validCookie)
	if err != nil {
		fmt.Printf("Ошибка верификации: %v\n", err)
		return
	}

	fmt.Printf("ID совпадают: %v\n", originalUserID == extractedUserID)

	// Output:
	// ID совпадают: true
}
