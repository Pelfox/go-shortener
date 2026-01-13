package main

import (
	"context"
	"os"

	"github.com/Pelfox/go-shortener/internal"
	"github.com/Pelfox/go-shortener/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
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
	storageLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().
		Timestamp().
		Str("component", "storage").
		Logger()

	ctx := context.Background()
	appConfig := internal.ParseAppConfig()

	var pool *pgxpool.Pool
	var err error

	if appConfig.DatabaseDSN != "" {
		pool, err = internal.NewDatabase(ctx, appConfig.DatabaseDSN)
		if err != nil {
			serverLogger.Fatal().Err(err).Msg("failed to connect to database")
		}
		defer pool.Close()
	}

	storage := internal.NewStorageFromConfig(storageLogger, appConfig.FilePath, pool)
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
