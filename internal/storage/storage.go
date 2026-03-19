package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/Pelfox/go-shortener/pkg"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

var (
	// ErrIDCollision указывает на попытку сохранить короткую ссылку с уже
	// существующим ID.
	ErrIDCollision = errors.New("redirect with given ID already exists")
	// ErrNotFound указывает на то, что короткая ссылка с запрашиваемым ID не найдена.
	ErrNotFound = errors.New("redirect with given ID not found")
	// ErrInvalidContext указывает на то, что не удалось найти пользовательский
	// ID в контексте.
	ErrInvalidContext = errors.New("the provided context is invalid")
	// ErrNotAuthorized указывает на то, что пользователь не имеет права на
	// выполнение данной операции над ссылкой.
	ErrNotAuthorized = errors.New("user is not an owner of the redirect")
	// ErrDeleted указывает на то, что данная ссылка была удалена.
	ErrDeleted = errors.New("redirect with given ID has been deleted")
)

// ShortenedLink - единичная репрезентация одной ссылки в хранилище.
type ShortenedLink struct {
	// ShortID это короткий ID для ссылки.
	ShortID string
	// OriginalURL это изначальный URL от пользователя.
	OriginalURL string
	// IsDeleted определяет, была ли удалена ссылка.
	IsDeleted bool
}

// Storage определяет интерфейс для хранения и получения коротких ссылок.
type Storage interface {
	// Store сохраняет короткую ссылку с заданным ID и URL назначения.
	Store(ctx context.Context, id string, destination string) error
	// Get возвращает URL назначения для короткой ссылки с заданным ID.
	Get(ctx context.Context, id string) (string, error)
	// GetByDestination возвращает ID короткой ссылки по исходному URL.
	GetByDestination(ctx context.Context, destination string) (string, error)
	// GetForUser возвращает все созданные этим пользователем сокращения.
	GetForUser(ctx context.Context) ([]ShortenedLink, error)
	// MarkDelete помечает ссылку для удаления.
	MarkDelete(ctx context.Context, userID string, shortIDs []string) error

	// Load загружает данные из системы хранения (файл, БД, пр.) в хранилище.
	Load() error
	// Save сохраняет данные из хранилища в систему хранения (файл, БД, пр.).
	Save() error
}

// NewStorageFromConfig выбирает PostgreSQL-хранилище при заданной строке
// подключения, иначе использует in-memory хранилище.
func NewStorageFromConfig(logger zerolog.Logger, filePath string, pool *pgxpool.Pool) Storage {
	logger = logger.With().Str("component", "storage").Logger()
	if pool == nil {
		logger.Info().Str("file", filePath).Msg("using in-memory storage")
		return NewInMemoryStorage(filePath)
	}
	logger.Info().Msg("using database-backed storage")
	return NewPostgresStorage(pool)
}

// redirect - объект, который используется InMemoryStorage для сохранения
// ссылок в память.
type redirect struct {
	// Destination - конечный URL пользователя.
	Destination string `json:"destination"`
	// UserID - ID пользователя, который создал эту переадресацию.
	UserID string `json:"user_id"`
	// IsDeleted - была ли удалена ссылка.
	IsDeleted bool `json:"is_deleted"`
}

// InMemoryStorage реализует интерфейс Storage, используя в памяти карту для
// хранения коротких ссылок.
type InMemoryStorage struct {
	mutex     sync.RWMutex
	redirects map[string]*redirect // ключ = ID для короткой ссылки, значение = исходная URL
	filePath  string
}

// NewInMemoryStorage создаёт новый экземпляр InMemoryStorage.
func NewInMemoryStorage(filePath string) *InMemoryStorage {
	return &InMemoryStorage{
		redirects: make(map[string]*redirect),
		filePath:  filePath,
	}
}

func (s *InMemoryStorage) MarkDelete(_ context.Context, userID string, shortIDs []string) error {
	// лишний раз не блокируем мьютекс
	if len(shortIDs) == 0 {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, shortID := range shortIDs {
		r, ok := s.redirects[shortID]
		if !ok {
			return ErrNotFound
		}

		if r.UserID != userID {
			return ErrNotAuthorized
		}

		r.IsDeleted = true
	}

	return nil
}

func (s *InMemoryStorage) GetForUser(ctx context.Context) ([]ShortenedLink, error) {
	userID, ok := ctx.Value(pkg.ContextUserIDKey).(string)
	if !ok {
		return nil, ErrInvalidContext
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	redirects := make([]ShortenedLink, 0, len(s.redirects))
	for k, v := range s.redirects {
		// не показываем удалённые ссылки
		if v.UserID == userID && !v.IsDeleted {
			redirects = append(redirects, ShortenedLink{
				ShortID:     k,
				OriginalURL: v.Destination,
			})
		}
	}

	return redirects, nil
}

func (s *InMemoryStorage) Store(ctx context.Context, id string, destination string) error {
	userID, ok := ctx.Value(pkg.ContextUserIDKey).(string)
	if !ok {
		return ErrInvalidContext
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.redirects[id]; exists {
		return ErrIDCollision
	}

	s.redirects[id] = &redirect{
		Destination: destination,
		UserID:      userID,
		IsDeleted:   false,
	}
	return nil
}

func (s *InMemoryStorage) Get(_ context.Context, id string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	r, exists := s.redirects[id]
	if !exists {
		return "", ErrNotFound
	}

	if r.IsDeleted {
		return "", ErrDeleted
	}

	return r.Destination, nil
}

func (s *InMemoryStorage) GetByDestination(_ context.Context, destination string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for id, dest := range s.redirects {
		if dest.Destination == destination {
			return id, nil
		}
	}

	return "", ErrNotFound
}

func (s *InMemoryStorage) Load() error {
	file, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // файл не существует, ничего загружать не нужно
		}
		return fmt.Errorf("could not open file %q: %w", s.filePath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("could not read file %q: %w", s.filePath, err)
	}

	if len(data) == 0 {
		return nil // файл пустой, ничего загружать не нужно
	}

	var fileData map[string]*redirect
	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("could not unmarshal data from file %q: %w", s.filePath, err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.redirects = fileData
	return nil
}

func (s *InMemoryStorage) Save() error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	data, err := json.Marshal(s.redirects)
	if err != nil {
		return fmt.Errorf("could not marshal data for file %q: %w", s.filePath, err)
	}

	if err := os.WriteFile(s.filePath, data, 0o644); err != nil {
		return fmt.Errorf("could not write data to file %q: %w", s.filePath, err)
	}

	return nil
}

// PostgresStorage реализует интерфейс Storage, используя PostgreSQL для
// хранения коротких ссылок.
type PostgresStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresStorage создаёт новый экземпляр PostgresStorage.
func NewPostgresStorage(pool *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{pool: pool}
}

func (s PostgresStorage) MarkDelete(ctx context.Context, userID string, shortIDs []string) error {
	if len(shortIDs) == 0 {
		return nil
	}
	_, err := s.pool.Exec(
		ctx,
		"UPDATE links SET is_deleted = true WHERE user_id = $1 AND slug = ANY($2) AND is_deleted = false",
		userID,
		shortIDs,
	)
	return err
}

func (s PostgresStorage) GetForUser(ctx context.Context) ([]ShortenedLink, error) {
	userID, ok := ctx.Value(pkg.ContextUserIDKey).(string)
	if !ok {
		return nil, ErrInvalidContext
	}

	rows, err := s.pool.Query(ctx,
		`SELECT slug, destination FROM links WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build query to get all user links: %w", err)
	}
	defer rows.Close()

	var links []ShortenedLink
	for rows.Next() {
		var link ShortenedLink
		err := rows.Scan(&link.ShortID, &link.OriginalURL)
		if err != nil {
			return nil, fmt.Errorf("failed to map user created link: %w", err)
		}
		links = append(links, link)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return links, nil
}

func (s PostgresStorage) Store(ctx context.Context, id string, destination string) error {
	userID, ok := ctx.Value(pkg.ContextUserIDKey).(string)
	if !ok {
		return ErrInvalidContext
	}

	_, err := s.pool.Exec(
		ctx,
		"INSERT INTO links (slug, destination, user_id) VALUES ($1, $2, $3)",
		id,
		destination,
		userID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		// 23505 - нарушение уникальности ключей
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrIDCollision
		}
		return fmt.Errorf("could not store link: %w", err)
	}
	return nil
}

func (s PostgresStorage) Get(ctx context.Context, id string) (string, error) {
	var destination string
	err := s.pool.QueryRow(ctx, "SELECT destination FROM links WHERE slug = $1", id).Scan(&destination)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("could not get link: %w", err)
	}
	return destination, nil
}

func (s PostgresStorage) GetByDestination(ctx context.Context, destination string) (string, error) {
	var slug string
	err := s.pool.QueryRow(ctx, "SELECT slug FROM links WHERE destination = $1", destination).Scan(&slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("could not get link by destination: %w", err)
	}
	return slug, nil
}

func (s PostgresStorage) Load() error {
	return nil
}

func (s PostgresStorage) Save() error {
	return nil
}
