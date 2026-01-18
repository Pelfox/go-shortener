package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Pelfox/go-shortener/internal"
	"github.com/Pelfox/go-shortener/internal/middlewares"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/Pelfox/go-shortener/pkg/schemas"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// maxGenerateAttempts определяет максимальное количество попыток сгенерировать
// уникальный короткий ID для ссылки.
const maxGenerateAttempts = 5

// errDestinationEmpty указывает на то, что переданный URL назначения пуст.
var errDestinationEmpty = errors.New("the destination URL is empty")

// Server реализует основной функционал приложения, а также HTTP-сервер.
type Server struct {
	addr    string
	baseURL string

	router *chi.Mux
	logger zerolog.Logger

	storage internal.Storage
	pool    *pgxpool.Pool
}

// NewServer создаёт и настраивает новый экземпляр Server.
func NewServer(
	addr string,
	baseURL string,
	logger zerolog.Logger,
	middlewareLogger zerolog.Logger,
	storage internal.Storage,
	pool *pgxpool.Pool,
) *Server {
	router := chi.NewRouter()
	router.Use(middlewares.LoggerMiddleware(middlewareLogger))
	router.Use(middlewares.CompressMiddleware)

	server := &Server{
		addr:    addr,
		baseURL: baseURL,
		router:  router,
		storage: storage,
		logger:  logger,
		pool:    pool,
	}

	router.Post("/", server.handleCreationRequest)
	router.Post("/api/shorten/batch", server.handleBatchRequest)
	router.Post("/api/shorten", server.handleShortenRequest)
	router.Get("/ping", server.handlePingRequest)
	router.Get("/*", server.handleShortRequest)

	return server
}

func (s *Server) createShortLink(ctx context.Context, destination string) (string, bool, error) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return "", false, errDestinationEmpty
	}

	existingID, err := s.storage.GetByDestination(ctx, destination)
	if err == nil {
		shortURL, err := url.JoinPath(s.baseURL, existingID)
		if err != nil {
			return "", false, err
		}
		return shortURL, true, nil
	}
	if !errors.Is(err, internal.ErrNotFound) {
		return "", false, err
	}

	for i := 0; i < maxGenerateAttempts; i++ {
		shortID := pkg.GenerateShortID(8)
		if err := s.storage.Store(ctx, shortID, destination); err != nil {
			if errors.Is(err, internal.ErrIDCollision) {
				continue // попытка снова при коллизии
			}
			return "", false, err
		}

		shortURL, err := url.JoinPath(s.baseURL, shortID)
		if err != nil {
			return "", false, err
		}

		return shortURL, false, nil
	}

	return "", false, errors.New("failed to generate a unique short ID")
}

func (s *Server) handleCreationRequest(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") {
		http.Error(w, "Invalid content type.", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to read body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	shortLink, conflict, err := s.createShortLink(r.Context(), string(body))
	if err != nil {
		if errors.Is(err, errDestinationEmpty) {
			http.Error(w, "The destination URL is empty.", http.StatusBadRequest)
			return
		}
		s.logger.Error().Err(err).Msg("failed to create short link")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if conflict {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	w.Write([]byte(shortLink))

	s.logger.Info().Str("slug", shortLink).
		Str("destination", string(body)).
		Msg("created short URL")
}

func (s *Server) handleShortenRequest(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Invalid content type.", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to read body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var request schemas.CreateShortLink
	if err := json.Unmarshal(body, &request); err != nil {
		s.logger.Error().Err(err).Msg("failed to unmarshal JSON")
		http.Error(w, "Invalid JSON body.", http.StatusBadRequest)
		return
	}

	shortLink, conflict, err := s.createShortLink(r.Context(), request.URL)
	if err != nil {
		if errors.Is(err, errDestinationEmpty) {
			http.Error(w, "The destination URL is empty.", http.StatusBadRequest)
			return
		}
		s.logger.Error().Err(err).Msg("failed to create short link")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response := schemas.ShortLinkResponse{Result: shortLink}
	responseBody, err := json.Marshal(response)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to marshal JSON")
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

	s.logger.Info().Str("slug", shortLink).
		Str("destination", request.URL).
		Msg("created short URL (via API)")
}

func (s *Server) handleShortRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	id := parts[len(parts)-1]
	destination, err := s.storage.Get(r.Context(), id)

	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			http.Error(w, "Short URL not found.", http.StatusNotFound)
			return
		}
		s.logger.Error().Err(err).Msg("failed to get short URL from storage")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, destination, http.StatusTemporaryRedirect)
}

func (s *Server) handlePingRequest(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		http.Error(w, "No pool available.", http.StatusServiceUnavailable)
		return
	}

	if err := s.pool.Ping(r.Context()); err != nil {
		s.logger.Error().Err(err).Msg("failed to ping database")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleBatchRequest(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Invalid content type.", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to read body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	s.logger.Info().
		Bytes("raw_body", body).
		Str("as_string", string(body)).
		Msg("DEBUG body")

	var request []schemas.BatchedLinkRequest
	if err := json.Unmarshal(body, &request); err != nil {
		s.logger.Error().Err(err).Msg("failed to unmarshal JSON111")
		http.Error(w, "Invalid JSON body.", http.StatusBadRequest)
		return
	}

	response := make([]schemas.BatchedLinkResponse, 0)
	hasConflicts := false
	for _, link := range request {
		shortLink, conflict, err := s.createShortLink(r.Context(), link.OriginalURL)
		if err != nil {
			if errors.Is(err, errDestinationEmpty) {
				http.Error(w, "The destination URL is empty.", http.StatusBadRequest)
				return
			}
			s.logger.Error().Err(err).Msg("failed to create short link")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if conflict {
			hasConflicts = true
		}
		response = append(response, schemas.BatchedLinkResponse{
			CorrelationID: link.CorrelationID,
			ShortURL:      shortLink,
		})
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to marshal JSON")
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

	s.logger.Info().Int("count", len(response)).
		Msg("created batched short links")
}

// ServeHTTP запускает HTTP-сервер и обрабатывает завершение работы.
func (s *Server) ServeHTTP() error {
	if err := s.storage.Load(); err != nil {
		return err
	}

	server := &http.Server{
		Addr:    s.addr,
		Handler: s.router,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		s.logger.Info().Str("addr", s.addr).Msg("starting server")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error().Err(err).Msg("caught a server error")
		}
	}()

	<-ctx.Done()
	if err := s.storage.Save(); err != nil {
		return err
	}

	return server.Shutdown(context.Background())
}
