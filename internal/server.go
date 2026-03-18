package internal

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Pelfox/go-shortener/internal/audit"
	"github.com/Pelfox/go-shortener/internal/handlers"
	"github.com/Pelfox/go-shortener/internal/middlewares"
	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Server struct {
	addr      string
	router    *chi.Mux
	logger    zerolog.Logger
	storage   storage.Storage
	providers []audit.Provider

	ctx    context.Context
	cancel context.CancelFunc
}

// NewServer создаёт и настраивает новый экземпляр Server.
func NewServer(
	config *AppConfig,
	logger zerolog.Logger,
	storageInstance storage.Storage,
	pool *pgxpool.Pool,
) *Server {
	userService := services.NewUserService(config.Secret)

	router := chi.NewRouter()
	router.Use(middlewares.LoggerMiddleware(logger))
	router.Use(middlewares.CompressMiddleware(logger))
	router.Use(middlewares.AuthMiddleware(userService, logger))

	providers := make([]audit.Provider, 0)
	if config.AuditURL != "" {
		providers = append(providers, audit.NewHTTPProvider(config.AuditURL))
	}
	if config.AuditFile != "" {
		providers = append(providers, audit.NewFileProvider(config.AuditFile))
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	server := &Server{
		addr:    config.Addr,
		router:  router,
		logger:  logger,
		storage: storageInstance,
		ctx:     ctx,
		cancel:  cancel,
	}

	// создаём новый ключевой сервис и запускаем фоновый очиститель
	shortenerService := services.NewShortenerService(
		ctx,
		config.BaseURL,
		storageInstance,
		logger,
		providers,
	)
	shortenerService.StartBackgroundCleaner()

	plainHandler := handlers.NewPlainHandler(shortenerService, logger)
	router.Post("/", plainHandler.Create)

	apiHandler := handlers.NewAPIHandler(shortenerService, logger)
	router.Route("/api", func(r chi.Router) {
		r.Post("/shorten", apiHandler.Create)
		r.Post("/shorten/batch", apiHandler.CreateBatch)
		r.Get("/user/urls", apiHandler.GetUserLinks)
		r.Delete("/user/urls", apiHandler.DeleteBatch)
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

	go func() {
		s.logger.Info().Str("addr", s.addr).Msg("starting server")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error().Err(err).Msg("caught a server error")
		}
	}()

	<-s.ctx.Done()
	s.cancel()
	if err := s.storage.Save(); err != nil {
		return err
	}

	return server.Shutdown(context.Background())
}
