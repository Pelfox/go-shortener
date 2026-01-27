package middlewares

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

// LoggerMiddleware логирует информацию о каждом HTTP-запросе.
func LoggerMiddleware(logger zerolog.Logger) func(handler http.Handler) http.Handler {
	logger = logger.With().Str("middleware", "logger").Logger()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrappedWriter := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			start := time.Now()
			next.ServeHTTP(wrappedWriter, r)
			duration := time.Since(start)

			method := r.Method
			path := r.RequestURI
			logger.Info().Str("method", method).
				Str("path", path).
				Int("status", wrappedWriter.Status()).
				Int("size", wrappedWriter.BytesWritten()).
				Dur("duration", duration).
				Msg("HTTP request processed")
		})
	}
}
