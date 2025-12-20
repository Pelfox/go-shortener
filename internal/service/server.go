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

	router  *chi.Mux
	storage internal.Storage
	logger  zerolog.Logger
}

// NewServer создаёт и настраивает новый экземпляр Server.
func NewServer(
	addr string,
	baseURL string,
	logger zerolog.Logger,
	middlewareLogger zerolog.Logger,
	storage internal.Storage,
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
	}

	router.Post("/", server.handleCreationRequest)
	router.Post("/api/shorten", server.handleShortenRequest)
	router.Get("/*", server.handleShortRequest)

	return server
}

func (s *Server) createShortLink(destination string) (string, error) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return "", errDestinationEmpty
	}

	for i := 0; i < maxGenerateAttempts; i++ {
		shortID := pkg.GenerateShortID(8)
		if err := s.storage.Store(shortID, destination); err != nil {
			if errors.Is(err, internal.ErrIDCollision) {
				continue // попытка снова при коллизии
			}
			return "", err
		}

		shortURL, err := url.JoinPath(s.baseURL, shortID)
		if err != nil {
			return "", err
		}

		return shortURL, nil
	}

	return "", errors.New("failed to generate a unique short ID")
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

	shortLink, err := s.createShortLink(string(body))
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
	w.WriteHeader(http.StatusCreated)
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

	shortLink, err := s.createShortLink(request.URL)
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
	w.WriteHeader(http.StatusCreated)
	w.Write(responseBody)

	s.logger.Info().Str("slug", shortLink).
		Str("destination", request.URL).
		Msg("created short URL (via API)")
}

func (s *Server) handleShortRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	id := parts[len(parts)-1]
	destination, err := s.storage.Get(id)

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
