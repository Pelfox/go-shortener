package pkg

import (
	"math/rand"
	"sync"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var (
	rng      = rand.New(rand.NewSource(time.Now().UnixNano()))
	rngMutex sync.Mutex
)

// GenerateShortID создаёт псевдослучайную строку заданной длины.
func GenerateShortID(length int) string {
	rngMutex.Lock()
	defer rngMutex.Unlock()

	result := make([]byte, length)
	for i := range result {
		result[i] = letters[rng.Intn(len(letters))]
	}

	return string(result)
}

// GenerateUserID генерирует псевдослучайную строку с ID пользователя, используя
// GenerateShortID "под капотом".
func GenerateUserID() string {
	return GenerateShortID(16)
}
