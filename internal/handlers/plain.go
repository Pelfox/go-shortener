package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/rs/zerolog"
)

// PlainHandler - обработчик базового API "сокращателя" (text/plain).
type PlainHandler struct {
	shortenerService *services.ShortenerService
	logger           zerolog.Logger
}

// NewPlainHandler создаёт новый PlainHandler с дочерним логгером.
func NewPlainHandler(
	shortenerService *services.ShortenerService,
	parentLogger zerolog.Logger,
) *PlainHandler {
	return &PlainHandler{
		shortenerService: shortenerService,
		logger:           parentLogger.With().Str("handler", "plain").Logger(),
	}
}

// Create создаёт новую короткую ссылку, читая из тела запроса конечный URL
// адрес для переадресации. Ожидаемый `Content-Type` - `text/plain`.
func (h *PlainHandler) Create(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "text/plain") {
		http.Error(w, "Invalid content type.", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to read the request body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	destination := string(body)
	shortURL, conflict, err := h.shortenerService.CreateShortLink(r.Context(), destination)

	if err != nil {
		if errors.Is(err, services.ErrDestinationEmpty) {
			http.Error(w, "The destination URL is empty.", http.StatusBadRequest)
			return
		}

		h.logger.Error().Err(err).Msg("failed to shorten the link")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if conflict {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	w.Write([]byte(shortURL))

	h.logger.Info().
		Str("shortURL", shortURL).
		Str("destination", destination).
		Msg("created a new shortened link")
}
