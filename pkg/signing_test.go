package pkg

import (
	"fmt"
	"testing"
)

// Тест для случая, когда был представлен корректный user ID и Cookie-содержимое.
func TestVerifyUserCookie_Valid(t *testing.T) {
	userID := "user123"
	secret := []byte("super-secret-key")

	signature := SignUserCookie(userID, secret)
	cookieValue := fmt.Sprintf("%s.%s", userID, signature)

	decodedUserID, ok := VerifyUserCookie(cookieValue, secret)
	if !ok {
		t.Fatalf("expected cookie to be valid")
	}

	if decodedUserID != userID {
		t.Fatalf("expected user ID %q, got %q", userID, decodedUserID)
	}
}

// Тест для случая, когда user ID подменяется внутри Cookie.
func TestVerifyUserCookie_TamperedUserID(t *testing.T) {
	userID := "user123"
	secret := []byte("super-secret-key")

	tamperedUserID := "admin"
	signature := SignUserCookie(userID, secret)
	cookieValue := fmt.Sprintf("%s.%s", tamperedUserID, signature)

	_, ok := VerifyUserCookie(cookieValue, secret)
	if ok {
		t.Fatalf("expected tampered user ID to be rejected")
	}
}

// Тест для случая, когда изменяется сигнатура Cookie содержимого.
func TestVerifyUserCookie_TamperedSignature(t *testing.T) {
	userID := "user123"
	secret := []byte("super-secret-key")

	signature := SignUserCookie(userID, secret)[1:]
	cookieValue := fmt.Sprintf("%s.%s", userID, signature)

	_, ok := VerifyUserCookie(cookieValue, secret)
	if ok {
		t.Fatalf("expected tampered signature to be rejected")
	}
}

// Тест для случая, когда подменяется секрет HMAC.
func TestVerifyUserCookie_WrongSecret(t *testing.T) {
	userID := "user123"

	correctSecret := []byte("super-secret-key")
	wrongSecret := []byte("wrong-secret")

	signature := SignUserCookie(userID, correctSecret)
	cookieValue := fmt.Sprintf("%s.%s", userID, signature)

	_, ok := VerifyUserCookie(cookieValue, wrongSecret)
	if ok {
		t.Fatalf("expected cookie signed with different secret to be rejected")
	}
}
