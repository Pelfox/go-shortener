package pkg

import (
	"math/rand"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateShortID создаёт псевдослучайную строку заданной длины.
func GenerateShortID(length int) string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := make([]byte, length)
	for i := range result {
		result[i] = letters[rng.Intn(len(letters))]
	}
	return string(result)
}
