package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	// ErrIDCollision указывает на попытку сохранить короткую ссылку с уже
	// существующим ID.
	ErrIDCollision = errors.New("redirect with given ID already exists")
	// ErrNotFound указывает на то, что короткая ссылка с запрашиваемым ID не найдена.
	ErrNotFound = errors.New("redirect with given ID not found")
)

// Storage определяет интерфейс для хранения и получения коротких ссылок.
type Storage interface {
	// Store сохраняет короткую ссылку с заданным ID и URL назначения.
	Store(id string, destination string) error
	// Get возвращает URL назначения для короткой ссылки с заданным ID.
	Get(id string) (string, error)

	// Load загружает данные из системы хранения (файл, БД, пр.) в хранилище.
	Load() error
	// Save сохраняет данные из хранилища в систему хранения (файл, БД, пр.).
	Save() error
}

// InMemoryStorage реализует интерфейс Storage, используя в памяти карту для
// хранения коротких ссылок.
type InMemoryStorage struct {
	mutex     *sync.RWMutex
	redirects map[string]string // ключ = ID для короткой ссылки, значение = исходная URL
	filePath  string
}

// NewInMemoryStorage создаёт новый экземпляр InMemoryStorage.
func NewInMemoryStorage(filePath string) *InMemoryStorage {
	return &InMemoryStorage{
		mutex:     &sync.RWMutex{},
		redirects: make(map[string]string),
		filePath:  filePath,
	}
}

func (s *InMemoryStorage) Store(id string, destination string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.redirects[id]; exists {
		return ErrIDCollision
	}

	s.redirects[id] = destination
	return nil
}

func (s *InMemoryStorage) Get(id string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	destination, exists := s.redirects[id]
	if !exists {
		return "", ErrNotFound
	}
	return destination, nil
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

	var fileData map[string]string
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

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("could not write data to file %q: %w", s.filePath, err)
	}

	return nil
}
