package main

import (
	"context"
	"os"

	"github.com/Pelfox/go-shortener/internal"
	"github.com/Pelfox/go-shortener/internal/service"
	"github.com/rs/zerolog"
)

func main() {
	serverLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().
		Timestamp().
		Str("component", "server").
		Logger()
	middlewareLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().
		Timestamp().
		Str("component", "logger-middleware").
		Logger()

	ctx := context.Background()
	appConfig := internal.ParseAppConfig()

	pool, err := internal.NewDatabase(ctx, appConfig.DatabaseDSN)
	if err != nil {
		serverLogger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	storage := internal.NewInMemoryStorage(appConfig.FilePath)
	server := service.NewServer(
		appConfig.Addr,
		appConfig.BaseURL,
		serverLogger,
		middlewareLogger,
		storage,
		pool,
	)

	if err := server.ServeHTTP(); err != nil {
		serverLogger.Fatal().Err(err).Msg("failed to start server")
	}
}
