package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// NewDatabase создаёт и возвращает подключение к базе данных PostgreSQL (pool).
func NewDatabase(ctx context.Context, databaseDSN string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, databaseDSN)
}

// RunMigrations применяет миграции из директории migrationsPath.
func RunMigrations(ctx context.Context, databaseDSN string, migrationsPath string) error {
	config, err := pgx.ParseConfig(databaseDSN)
	if err != nil {
		return fmt.Errorf("could not parse database dsn: %w", err)
	}

	db := stdlib.OpenDB(*config)
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create migrate driver: %w", err)
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("could not resolve migrations path: %w", err)
	}

	sourceURL := fmt.Sprintf("file://%s", absPath)
	migrator, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return fmt.Errorf("could not create migrator: %w", err)
	}
	defer func() {
		_, _ = migrator.Close()
	}()

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("could not apply migrations: %w", err)
	}

	return nil
}
