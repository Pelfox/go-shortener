package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// HealthHandler - обработчик запросов, связанных со "здоровьем" приложения.
type HealthHandler struct {
	pool   *pgxpool.Pool // TODO: возможно, стоит добавить сюда более общую абстракцию.
	logger zerolog.Logger
}

// NewHealthHandler создаёт и возвращает новый HealthHandler с дочерним логгером.
func NewHealthHandler(
	pool *pgxpool.Pool,
	parentLogger zerolog.Logger,
) *HealthHandler {
	return &HealthHandler{
		pool:   pool,
		logger: parentLogger.With().Str("handler", "health").Logger(),
	}
}

// Ping проверяет работоспособность и готовность пула базы данных.
func (h *HealthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		http.Error(w, "No pool available.", http.StatusServiceUnavailable)
		return
	}

	if err := h.pool.Ping(r.Context()); err != nil {
		h.logger.Error().Err(err).Msg("failed to ping the database pool")
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}
