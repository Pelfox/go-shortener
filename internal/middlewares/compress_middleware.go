package middlewares

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer

	gzipEnabled bool // включён ли gzip для данного запроса
	wroteHeader bool // записали ли заголовки для gzip
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.wroteHeader = true
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html") {

		w.gzipEnabled = true
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")

		w.writer = gzip.NewWriter(w.ResponseWriter)
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	// реализуем стандартный функционал из Go: пишем 200 OK если обработчик ещё
	// не записал заголовки в ответ
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.gzipEnabled {
		return w.writer.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *gzipResponseWriter) Close() error {
	if w.writer != nil {
		return w.writer.Close()
	}
	return nil
}

// CompressMiddleware компрессирует ответы и декомпрессирует запросы.
func CompressMiddleware(logger zerolog.Logger) func(handler http.Handler) http.Handler {
	logger = logger.With().Str("middleware", "compress").Logger()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// получили запрос с gzip-содержимым в теле
			if r.Header.Get("Content-Encoding") == "gzip" {
				gzipReader, err := gzip.NewReader(r.Body)
				if err != nil {
					logger.Error().Err(err).Msg("failed to read gzip-compressed body")
					http.Error(w, "Invalid body.", http.StatusBadRequest)
					return
				}
				defer gzipReader.Close()
				r.Body = io.NopCloser(gzipReader)
			}

			// клиент не поддерживает компрессию gzip
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				logger.Info().Msg("client does not support gzip")
				next.ServeHTTP(w, r)
				return
			}

			writer := &gzipResponseWriter{ResponseWriter: w}
			defer writer.Close()
			next.ServeHTTP(writer, r)
			logger.Info().Msg("serving the request with gzip compression")
		})
	}
}
