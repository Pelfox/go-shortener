package internal

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Pelfox/go-shortener/internal/audit"
	grpcserver "github.com/Pelfox/go-shortener/internal/grpc"
	"github.com/Pelfox/go-shortener/internal/handlers"
	"github.com/Pelfox/go-shortener/internal/middlewares"
	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Server struct {
	addr            string
	grpcAddr        string
	enableHTTPS     bool
	enableGRPCTLS   bool
	router          *chi.Mux
	logger          zerolog.Logger
	storage         storage.Storage
	providers       []audit.Provider
	shortenerServer *services.ShortenerService
	userService     *services.UserService

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
		os.Interrupt, // Эквивалентно syscall.SIGINT
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	server := &Server{
		addr:          config.Addr,
		grpcAddr:      config.GRPCAddr,
		enableHTTPS:   config.EnableHTTPS,
		enableGRPCTLS: config.EnableGRPCTLS,
		router:        router,
		logger:        logger,
		storage:       storageInstance,
		providers:     providers,
		userService:   userService,
		ctx:           ctx,
		cancel:        cancel,
	}
	if server.grpcAddr == "" {
		server.grpcAddr = "localhost:3200"
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
	server.shortenerServer = shortenerService

	plainHandler := handlers.NewPlainHandler(shortenerService, logger)
	router.Post("/", plainHandler.Create)

	apiHandler := handlers.NewAPIHandler(
		shortenerService,
		logger,
		config.TrustedSubnet,
		userService,
	)
	router.Route("/api", func(r chi.Router) {
		r.Post("/shorten", apiHandler.Create)
		r.Post("/shorten/batch", apiHandler.CreateBatch)
		r.Get("/user/urls", apiHandler.GetUserLinks)
		r.Get("/internal/stats", apiHandler.GetStats)
		r.Delete("/user/urls", apiHandler.DeleteBatch)
	})

	healthHandler := handlers.NewHealthHandler(pool, logger)
	router.Get("/ping", healthHandler.Ping)

	redirectHandler := handlers.NewRedirectHandler(shortenerService, logger)
	router.Get("/{id}", redirectHandler.Redirect)

	return server
}

// Serve запускает HTTP и gRPC серверы и обрабатывает завершение работы.
func (s *Server) Serve() error {
	if err := s.storage.Load(); err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:    s.addr,
		Handler: s.router,
	}

	grpcOptions := make([]googlegrpc.ServerOption, 0, 1)
	if s.enableGRPCTLS {
		if err := validateCertFiles("cert.pem", "key.pem"); err != nil {
			return err
		}

		creds, err := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
		if err != nil {
			return fmt.Errorf("failed to create gRPC TLS credentials: %w", err)
		}
		grpcOptions = append(grpcOptions, googlegrpc.Creds(creds))
	}

	grpcServer := grpcserver.NewServer(
		s.shortenerServer,
		s.userService,
		s.logger,
		grpcOptions...,
	)
	grpcListener, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC address %q: %w", s.grpcAddr, err)
	}

	errChan := make(chan error, 2)

	go func() {
		s.logger.Info().Str("addr", s.addr).Msg("starting server")
		var err error
		if s.enableHTTPS {
			if err = validateCertFiles("cert.pem", "key.pem"); err != nil {
				errChan <- fmt.Errorf("HTTP TLS certificates validation failed: %w", err)
				return
			}
			err = httpServer.ListenAndServeTLS("cert.pem", "key.pem")
		} else {
			err = httpServer.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("caught an HTTP server error: %w", err)
		}
	}()

	go func() {
		s.logger.Info().Str("addr", s.grpcAddr).Msg("starting gRPC server")
		if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, googlegrpc.ErrServerStopped) {
			errChan <- fmt.Errorf("caught a gRPC server error: %w", err)
		}
	}()

	var serveErr error
	select {
	case <-s.ctx.Done():
	case serveErr = <-errChan:
		s.logger.Error().Err(serveErr).Msg("server stopped with error")
	}

	s.cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	httpErr := httpServer.Shutdown(shutdownCtx)

	grpcStopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcStopped)
	}()
	select {
	case <-grpcStopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}

	saveErr := s.storage.Save()

	if serveErr != nil {
		return serveErr
	}
	if httpErr != nil {
		return httpErr
	}
	if saveErr != nil {
		return saveErr
	}

	return nil
}

// validateCertFiles проверяет наличие и доступность TLS-сертификата и ключа.
func validateCertFiles(certFile, keyFile string) error {
	if _, err := os.Stat(certFile); err != nil {
		return fmt.Errorf("failed to access certificate file %q: %w", certFile, err)
	}

	if _, err := os.Stat(keyFile); err != nil {
		return fmt.Errorf("failed to access private key file %q: %w", keyFile, err)
	}

	return nil
}
