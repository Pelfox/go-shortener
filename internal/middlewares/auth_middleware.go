package middlewares

import (
	"context"
	"net/http"

	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/rs/zerolog"
)

// cookieName - название Cookie для пользовательского ID и сигнатуры.
const cookieName = "user_id"

// AuthMiddleware реализует базовую проверку на Cookie с ID пользователя, а
// также валидирует её через HMAC.
func AuthMiddleware(
	userService *services.UserService,
	logger zerolog.Logger,
) func(http.Handler) http.Handler {
	logger = logger.With().Str("middleware", "auth").Logger()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)

			// если кука была найдена
			if err == nil {
				userID, err := userService.VerifyUserCookieValue(cookie.Value)

				// кука валидна, ставим пользовательский ID в контекст запроса
				if err == nil {
					userContext := context.WithValue(r.Context(), pkg.ContextUserIDKey, userID)
					next.ServeHTTP(w, r.WithContext(userContext))
					return
				}

				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			userID, cookieValue := userService.CreateUserCookie()
			http.SetCookie(w, &http.Cookie{
				Name:  cookieName,
				Value: cookieValue,
			})
			userContext := context.WithValue(r.Context(), pkg.ContextUserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(userContext))
		})
	}
}
