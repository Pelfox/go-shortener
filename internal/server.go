package internal

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Pelfox/go-shortener/internal/handlers"
	"github.com/Pelfox/go-shortener/internal/middlewares"
	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Server struct {
	addr    string
	router  *chi.Mux
	logger  zerolog.Logger
	storage storage.Storage
}

// NewServer создаёт и настраивает новый экземпляр Server.
func NewServer(
	addr string,
	baseURL string,
	logger zerolog.Logger,
	secret []byte,
	storageInstance storage.Storage,
	pool *pgxpool.Pool,
) *Server {
	userService := services.NewUserService(secret)

	router := chi.NewRouter()
	router.Use(middlewares.LoggerMiddleware(logger))
	router.Use(middlewares.CompressMiddleware(logger))
	router.Use(middlewares.AuthMiddleware(userService, logger))

	server := &Server{
		addr:    addr,
		router:  router,
		logger:  logger,
		storage: storageInstance,
	}
	shortenerService := services.NewShortenerService(baseURL, storageInstance)

	plainHandler := handlers.NewPlainHandler(shortenerService, logger)
	router.Post("/", plainHandler.Create)

	apiHandler := handlers.NewAPIHandler(shortenerService, logger)
	router.Route("/api", func(r chi.Router) {
		r.Post("/shorten", apiHandler.Create)
		r.Post("/shorten/batch", apiHandler.CreateBatch)
		r.Get("/user/urls", apiHandler.GetUserLinks)
	})

	healthHandler := handlers.NewHealthHandler(pool, logger)
	router.Get("/ping", healthHandler.Ping)

	redirectHandler := handlers.NewRedirectHandler(shortenerService, logger)
	router.Get("/{id}", redirectHandler.Redirect)

	return server
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
