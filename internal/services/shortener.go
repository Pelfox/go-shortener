package services

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/Pelfox/go-shortener/pkg/schemas"
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
)

// ShortenerService реализует логику сокращения и хранения ссылок.
type ShortenerService struct {
	baseURL string
	storage storage.Storage
}

// NewShortenerService создаёт и возвращает новый экземпляр сервиса сокращения ссылок.
func NewShortenerService(baseURL string, storage storage.Storage) *ShortenerService {
	return &ShortenerService{
		baseURL: baseURL,
		storage: storage,
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
