package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/Pelfox/go-shortener/pkg/schemas"
	"github.com/rs/zerolog"
)

// APIHandler - обработчик API "сокращателя" с JSON содержимым.
type APIHandler struct {
	shortenerService *services.ShortenerService
	logger           zerolog.Logger
}

// NewAPIHandler создаёт новый APIHandler с дочерним логгером.
func NewAPIHandler(
	shortenerService *services.ShortenerService,
	parentLogger zerolog.Logger,
) *APIHandler {
	return &APIHandler{
		shortenerService: shortenerService,
		logger:           parentLogger.With().Str("handler", "api").Logger(),
	}
}

// Create создаёт новую короткую ссылку, читая запрос из тела. Ожидаемый
// `Content-Type` - `application/json`.
func (h *APIHandler) Create(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
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

	var request schemas.CreateShortLink
	if err := json.Unmarshal(body, &request); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse the request body")
		http.Error(w, "Invalid body.", http.StatusBadRequest)
		return
	}

	shortURL, conflict, err := h.shortenerService.CreateShortLink(r.Context(), request.URL)
	if err != nil {
		if errors.Is(err, services.ErrDestinationEmpty) {
			http.Error(w, "The destination URL is empty.", http.StatusBadRequest)
			return
		}

		h.logger.Error().Err(err).Msg("failed to shorten the link")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response := schemas.ShortLinkResponse{Result: shortURL}
	responseBody, err := json.Marshal(response)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal the response body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if conflict {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	w.Write(responseBody)

	h.logger.Info().
		Str("shortURL", shortURL).
		Str("destination", request.URL).
		Msg("created a new shortened link")
}

// CreateBatch создаёт множество коротких ссылок для каждой ссылки в теле
// запроса. Ожидаемый `Content-Type` - `application/json`.
func (h *APIHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
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

	var requests []schemas.BatchedLinkRequest
	if err := json.Unmarshal(body, &requests); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse the request body")
		http.Error(w, "Invalid body.", http.StatusBadRequest)
		return
	}

	hasConflicts := false
	responses := make([]schemas.BatchedLinkResponse, 0)

	for _, request := range requests {
		shortURL, conflict, err := h.shortenerService.CreateShortLink(r.Context(), request.OriginalURL)
		if err != nil {
			if errors.Is(err, services.ErrDestinationEmpty) {
				http.Error(w, "The destination URL is empty.", http.StatusBadRequest)
				return
			}

			h.logger.Error().Err(err).Msg("failed to shorten the link")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if conflict {
			hasConflicts = true
		}
		responses = append(responses, schemas.BatchedLinkResponse{
			CorrelationID: request.CorrelationID,
			ShortURL:      shortURL,
		})
	}

	responseBody, err := json.Marshal(responses)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal the response body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if hasConflicts {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	w.Write(responseBody)

	h.logger.Info().
		Int("count", len(responses)).
		Msg("processed a batched link creation request")
}

// GetUserLinks возвращает все ссылки, созданные данным пользователем.
func (h *APIHandler) GetUserLinks(w http.ResponseWriter, r *http.Request) {
	links, err := h.shortenerService.GetUserLinks(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get user links")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(links) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	responseBody, err := json.Marshal(links)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal the response body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBody)
}

// DeleteBatch удаляет все указанные ссылки (асинхронно, через fan-in).
func (h *APIHandler) DeleteBatch(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
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

	var shortIDs []string
	if err := json.Unmarshal(body, &shortIDs); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse the request body")
		http.Error(w, "Invalid body.", http.StatusBadRequest)
		return
	}

	if err := h.shortenerService.DeleteBatch(r.Context(), shortIDs); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete shortened links")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
