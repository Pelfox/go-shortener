package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Pelfox/go-shortener/internal"
	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// printBuildInfo выводит информацию о текущей сборке.
func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}
	date := buildDate
	if date == "" {
		date = "N/A"
	}
	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}

	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}

func main() {
	printBuildInfo()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	ctx := context.Background()
	appConfig := internal.ParseAppConfig()

	var (
		pool *pgxpool.Pool
		err  error
	)

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
	server := internal.NewServer(appConfig, logger, storageInstance, pool)

	if err := server.ServeHTTP(); err != nil {
		logger.Fatal().Err(err).Msg("failed to start the server")
	}
}
