package internal

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDatabase создаёт и возвращает подключение к базе данных PostgreSQL (pool).
func NewDatabase(ctx context.Context, databaseDSN string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, databaseDSN)
}
