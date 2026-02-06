package handlers

import (
	"errors"
	"net/http"

	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// RedirectHandler - обработчик запросов, связанных с переадресацией пользователя.
type RedirectHandler struct {
	shortenerService *services.ShortenerService
	logger           zerolog.Logger
}

// NewRedirectHandler создаёт и возвращает RedirectHandler с дочерним логгером.
func NewRedirectHandler(
	shortenerService *services.ShortenerService,
	parentLogger zerolog.Logger,
) *RedirectHandler {
	return &RedirectHandler{
		shortenerService: shortenerService,
		logger:           parentLogger.With().Str("handler", "redirect").Logger(),
	}
}

// Redirect переадресует пользователя на конечный URL, привязанный к этому
// короткому ID, получаемому из запроса.
func (h *RedirectHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "id")

	destination, err := h.shortenerService.GetDestination(r.Context(), shortID)
	if err != nil {
		if errors.Is(err, services.ErrShortIDEmpty) {
			http.Error(w, "Missing short ID", http.StatusBadRequest)
			return
		}
		if errors.Is(err, services.ErrDestinationNotFound) {
			http.Error(w, "Destination not found", http.StatusNotFound)
			return
		}
		h.logger.Error().Err(err).Msg("failed to process the redirection")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, destination, http.StatusTemporaryRedirect)
}
