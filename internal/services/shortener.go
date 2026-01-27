package services

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/Pelfox/go-shortener/pkg/schemas"
	"github.com/rs/zerolog"
)

// maxGenerateAttempts указывает на максимальное количество попыток генерации
// уникального короткого ID для ссылки.
const maxGenerateAttempts = 3

var (
	// ErrDestinationEmpty указывает, что передан пустой адрес назначения ссылки.
	ErrDestinationEmpty = errors.New("destination is empty")
	// ErrShortIDGenerationFailed указывает, что не удалось сгенерировать
	// уникальный короткий ID после `maxGenerateAttempts` попыток генерации.
	ErrShortIDGenerationFailed = errors.New("short ID generation failed")
	// ErrShortIDEmpty указывает, что предоставленный короткий ID ссылки пуст.
	ErrShortIDEmpty = errors.New("short ID is empty")
	// ErrDestinationNotFound указывает, что для данного короткого ID не
	// существует финальной ссылки.
	ErrDestinationNotFound = errors.New("no destination for this short ID")
	// ErrDeleted указывает на то что данная ссылка была помечена как удалённая.
	ErrDeleted = errors.New("this link has been deleted")
)

type deleteTask struct {
	UserID   string
	ShortIDs []string
}

// ShortenerService реализует логику сокращения ссылок.
type ShortenerService struct {
	ctx        context.Context
	baseURL    string
	storage    storage.Storage
	deleteChan chan deleteTask
	logger     zerolog.Logger
}

// NewShortenerService создаёт и возвращает новый экземпляр сервиса сокращения ссылок.
func NewShortenerService(
	ctx context.Context,
	baseURL string,
	storage storage.Storage,
	parentLogger zerolog.Logger,
) *ShortenerService {
	return &ShortenerService{
		ctx:        ctx,
		baseURL:    baseURL,
		storage:    storage,
		deleteChan: make(chan deleteTask, 50), // TODO: должно ли это быть настраиваемым через конфиг?
		logger:     parentLogger.With().Str("service", "shortener").Logger(),
	}
}

// GetDestination возвращает исходный URL, привязанный к данному короткому ID
// из базы данных.
func (s *ShortenerService) GetDestination(ctx context.Context, shortID string) (string, error) {
	shortID = strings.TrimSpace(shortID)
	if len(shortID) == 0 {
		return "", ErrShortIDEmpty
	}

	destination, err := s.storage.Get(ctx, shortID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return "", ErrDestinationNotFound
		}
		if errors.Is(err, storage.ErrDeleted) {
			return "", ErrDeleted
		}
		return "", err
	}

	return destination, nil
}

// CreateShortLink сокращает переданную ссылку. Если ссылка уже была сокращена -
// возвращает существующий адрес (второй параметр = true). Иначе генерирует
// новый короткий ID и сохраняет пару в хранилище.
func (s *ShortenerService) CreateShortLink(ctx context.Context, destination string) (string, bool, error) {
	destination = strings.TrimSpace(destination)
	if len(destination) == 0 {
		return "", false, ErrDestinationEmpty
	}

	// проверяем, существует ли уже короткий ID для данной ссылки
	existingID, err := s.storage.GetByDestination(ctx, destination)
	if err == nil {
		// соединяем базовый URL из конфига и короткий ID
		shortURL, err := url.JoinPath(s.baseURL, existingID)
		if err != nil {
			return "", false, err
		}

		return shortURL, true, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return "", false, err
	}

	for i := 0; i < maxGenerateAttempts; i++ {
		shortID := pkg.GenerateShortID(8)
		if err := s.storage.Store(ctx, shortID, destination); err != nil {
			// если получаем коллизию - пробуем ещё раз
			if errors.Is(err, storage.ErrIDCollision) {
				continue
			}
			return "", false, err
		}

		// соединяем базовый URL из конфига и короткий ID
		shortURL, err := url.JoinPath(s.baseURL, shortID)
		if err != nil {
			return "", false, err
		}

		return shortURL, false, nil
	}

	return "", false, ErrShortIDGenerationFailed
}

// GetUserLinks возвращает все ссылки, созданные данным пользователем.
func (s *ShortenerService) GetUserLinks(ctx context.Context) ([]schemas.ShortenedUserLink, error) {
	links, err := s.storage.GetForUser(ctx)
	if err != nil {
		return nil, err
	}

	userLinks := make([]schemas.ShortenedUserLink, 0)
	for _, link := range links {
		// соединяем базовый URL из конфига и короткий ID
		shortURL, err := url.JoinPath(s.baseURL, link.ShortID)
		if err != nil {
			return nil, err
		}

		userLinks = append(userLinks, schemas.ShortenedUserLink{
			ShortURL:    shortURL,
			OriginalURL: link.OriginalURL,
		})
	}

	return userLinks, nil
}

func (s *ShortenerService) processBatchedDeletion(batchTasks []deleteTask) {
	for _, task := range batchTasks {
		if err := s.storage.MarkDelete(s.ctx, task.UserID, task.ShortIDs); err != nil {
			s.logger.Error().Err(err).
				Str("user", task.UserID).
				Msg("failed to mark link as deleted")
		}
	}
}

// StartBackgroundCleaner запускает goroutine, которая последовательно очищает
// накопившиеся запросы на удаление ссылок, либо каждые 10 запросов, либо раз в
// 500 миллисекунд.
func (s *ShortenerService) StartBackgroundCleaner() {
	const (
		maxBatchSize = 10
		flushTimeout = 500 * time.Millisecond
	)

	s.logger.Info().Int("maxBatchSize", maxBatchSize).
		Dur("flushTimeout", flushTimeout).
		Msg("starting background cleaner")

	batchTasks := make([]deleteTask, 0, maxBatchSize)
	ticker := time.NewTicker(flushTimeout)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case task := <-s.deleteChan:
				batchTasks = append(batchTasks, task)
				if len(batchTasks) >= maxBatchSize {
					s.processBatchedDeletion(batchTasks)
					batchTasks = batchTasks[:0]
				}
			case <-ticker.C:
				if len(batchTasks) > 0 {
					s.processBatchedDeletion(batchTasks)
					batchTasks = batchTasks[:0]
				}
			case <-s.ctx.Done():
				// дообрабатываем все оставшиеся запросы на удаление
				s.processBatchedDeletion(batchTasks)
				return
			}
		}
	}()
}

// DeleteBatch отправляет запрос на удаление сразу нескольких коротких ID ссылок.
func (s *ShortenerService) DeleteBatch(ctx context.Context, shortIDs []string) error {
	userID, ok := ctx.Value(pkg.ContextUserIDKey).(string)
	if !ok {
		return storage.ErrInvalidContext
	}

	select {
	case s.deleteChan <- deleteTask{
		UserID:   userID,
		ShortIDs: shortIDs,
	}:
		return nil
	default:
		s.logger.Warn().Msg("delete queue is full, dropping task")
	}
	return nil
}
