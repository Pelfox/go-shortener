package middlewares

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/rs/zerolog"
)

func prepareAuthLogger(t *testing.T) zerolog.Logger {
	t.Helper()
	return zerolog.New(io.Discard)
}

func prepareUserService(t *testing.T) *services.UserService {
	t.Helper()
	return services.NewUserService([]byte("test-secret"))
}

// Тест для случая, когда куки нет, и необходимо создать новый пользовательский ID.
func TestAuthMiddleware_NoCookie(t *testing.T) {
	logger := prepareAuthLogger(t)
	userService := prepareUserService(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(pkg.ContextUserIDKey)
		if userID == nil {
			t.Fatal("expected userID in context, got nil")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	AuthMiddleware(userService, logger)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	if cookies[0].Name != cookieName {
		t.Fatalf("expected cookie name %q, got %q", cookieName, cookies[0].Name)
	}
}

// Тест для случая, когда был найден корректная кука.
func TestAuthMiddleware_ValidCookie(t *testing.T) {
	logger := prepareAuthLogger(t)
	userService := prepareUserService(t)

	userID, cookieValue := userService.CreateUserCookie()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID := r.Context().Value(pkg.ContextUserIDKey)
		if ctxUserID != userID {
			t.Fatalf("expected userID %q, got %v", userID, ctxUserID)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: cookieValue,
	})

	rec := httptest.NewRecorder()
	AuthMiddleware(userService, logger)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

// Тест для случая, когда была найдена невалидная кука.
func TestAuthMiddleware_InvalidCookie(t *testing.T) {
	logger := prepareAuthLogger(t)
	userService := prepareUserService(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called on invalid cookie")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: "invalid.cookie.value",
	})

	rec := httptest.NewRecorder()
	AuthMiddleware(userService, logger)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}
}

// Тест для случая, когда парсинг удался и кука валидна.
func TestAuthMiddleware_CallsNextHandler(t *testing.T) {
	logger := prepareAuthLogger(t)
	userService := prepareUserService(t)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	AuthMiddleware(userService, logger)(next).ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected next handler to be called")
	}
}
