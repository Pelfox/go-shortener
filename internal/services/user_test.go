package services

import (
	"errors"
	"strings"
	"testing"
)

func prepareUserService(t *testing.T) *UserService {
	t.Helper()
	return NewUserService([]byte("test-secret"))
}

// Тест создания новой пользовательской cookie.
func TestUserService_CreateUserCookie(t *testing.T) {
	service := prepareUserService(t)
	userID, cookieValue := service.CreateUserCookie()

	if userID == "" {
		t.Fatal("expected non-empty userID")
	}

	if cookieValue == "" {
		t.Fatal("expected non-empty cookieValue")
	}

	// cookie должен содержать userID и подпись, разделённые точкой
	if !strings.HasPrefix(cookieValue, userID+".") {
		t.Fatalf("cookie value must start with userID, got %q", cookieValue)
	}
}

// Тест успешной валидации корректной cookie.
func TestUserService_VerifyUserCookieValue_Valid(t *testing.T) {
	service := prepareUserService(t)
	userID, cookieValue := service.CreateUserCookie()

	verifiedID, err := service.VerifyUserCookieValue(cookieValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if verifiedID != userID {
		t.Fatalf("expected userID %q, got %q", userID, verifiedID)
	}
}

// Тест невалидной cookie при подмене userID.
func TestUserService_VerifyUserCookieValue_TamperedUserID(t *testing.T) {
	service := prepareUserService(t)
	userID, cookieValue := service.CreateUserCookie()

	// подменяем userID, сохраняя старую подпись
	tamperedCookie := "hacker-id" + cookieValue[len(userID):]

	_, err := service.VerifyUserCookieValue(tamperedCookie)
	if !errors.Is(err, ErrCookieInvalid) {
		t.Fatalf("expected ErrCookieInvalid, got %v", err)
	}
}

// Тест невалидной cookie при подмене подписи.
func TestUserService_VerifyUserCookieValue_TamperedSignature(t *testing.T) {
	service := prepareUserService(t)
	userID, _ := service.CreateUserCookie()

	// корректный userID, но поддельная подпись
	tamperedCookie := userID + ".invalid-signature"

	_, err := service.VerifyUserCookieValue(tamperedCookie)
	if !errors.Is(err, ErrCookieInvalid) {
		t.Fatalf("expected ErrCookieInvalid, got %v", err)
	}
}

// Тест невалидной cookie с некорректным форматом.
func TestUserService_VerifyUserCookieValue_InvalidFormat(t *testing.T) {
	service := prepareUserService(t)
	_, err := service.VerifyUserCookieValue("not-a-valid-cookie")
	if !errors.Is(err, ErrCookieInvalid) {
		t.Fatalf("expected ErrCookieInvalid, got %v", err)
	}
}
