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

// BuildInfo описывает информацию о текущей сборке программы.
type BuildInfo struct {
	// Version это текущая версия программы.
	Version string
	// Date это время, когда была осуществлена данная сборка.
	Date string
	// Commit это хэш коммита, который привязан к данной сборке.
	Commit string
}

// printBuildInfo выводит информацию о текущей сборке.
func printBuildInfo(info BuildInfo) {
	version := info.Version
	if version == "" {
		version = "N/A"
	}
	date := info.Date
	if date == "" {
		date = "N/A"
	}
	commit := info.Commit
	if commit == "" {
		commit = "N/A"
	}

	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}

func main() {
	printBuildInfo(BuildInfo{
		Version: buildVersion,
		Date:    buildDate,
		Commit:  buildCommit,
	})

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	ctx := context.Background()

	appConfig, err := internal.ParseAppConfig(logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to parse configuration")
	}

	var pool *pgxpool.Pool
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
