package internal

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	addr        string
	enableHTTPS bool
	router      *chi.Mux
	logger      zerolog.Logger
	storage     storage.Storage
	providers   []audit.Provider

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
		addr:        config.Addr,
		enableHTTPS: config.EnableHTTPS,
		router:      router,
		logger:      logger,
		storage:     storageInstance,
		ctx:         ctx,
		cancel:      cancel,
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
		var err error
		if s.enableHTTPS {
			if err = ensureCerts("cert.pem", "key.pem"); err != nil {
				s.logger.Error().Err(err).Msg("failed to generate certs")
			}
			err = server.ListenAndServeTLS("cert.pem", "key.pem")
		} else {
			err = server.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
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

// ensureCerts проверяет наличие сертификатов и генерирует их в случае отсутствия.
func ensureCerts(certFile, keyFile string) error {
	if _, err := os.Stat(certFile); err == nil {
		return nil
	}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{"Yandex.Practicum"},
			Country:      []string{"RU"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate a private key: %w", err)
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create the certificate: %w", err)
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}
	defer keyOut.Close()
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return nil
}
