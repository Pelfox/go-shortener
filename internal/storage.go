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
	ErrIDCollision = errors.New("redirect with given ID already exists")
	ErrNotFound    = errors.New("redirect with given ID not found")
)

type Storage interface {
	Store(id string, destination string) error
	Get(id string) (string, error)

	Load() error
	Save() error
}

type InMemoryStorage struct {
	mutex     *sync.RWMutex
	redirects map[string]string // ключ = ID для короткой ссылки, значение = исходная URL
	filePath  string
}

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
