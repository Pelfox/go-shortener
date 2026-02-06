package main

import (
	"context"
	"os"

	"github.com/Pelfox/go-shortener/internal"
	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	ctx := context.Background()
	appConfig := internal.ParseAppConfig()

	var pool *pgxpool.Pool
	var err error

	if appConfig.DatabaseDSN != "" {
		if err := storage.RunMigrations(ctx, appConfig.DatabaseDSN, "migrations"); err != nil {
			logger.Fatal().Err(err).Msg("failed to apply migrations")
		}

		pool, err = storage.NewDatabase(ctx, appConfig.DatabaseDSN)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to connect to database")
		}
		defer pool.Close()
	}

	storageInstance := storage.NewStorageFromConfig(logger, appConfig.FilePath, pool)
	server := internal.NewServer(
		appConfig.Addr,
		appConfig.BaseURL,
		logger,
		appConfig.Secret,
		storageInstance,
		pool,
	)

	if err := server.ServeHTTP(); err != nil {
		logger.Fatal().Err(err).Msg("failed to start the server")
	}
}
