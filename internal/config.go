package internal

import (
	"flag"
	"os"
)

// AppConfig хранит конфигурацию приложения.
type AppConfig struct {
	// Addr это адрес для HTTP-сервера.
	Addr string
	// BaseURL это базовый URL для коротких ссылок.
	BaseURL string
	// FilePath это путь к файлу для хранения ссылок.
	FilePath string
}

func getConfigValue(envName string, flagValue *string) string {
	// сначала проверяем переменную окружения
	envValue, ok := os.LookupEnv(envName)
	if ok {
		return envValue
	}
	// пытаемся взять значение из флага, или возвращаем значение по умолчанию
	// (flag.String уже содержит значение по умолчанию)
	return *flagValue
}

// ParseAppConfig парсит конфигурацию приложения из флагов командной строки и
// переменных окружения.
func ParseAppConfig() *AppConfig {
	hostFlag := flag.String("a", "localhost:8080", "Адрес, на котором будет запущен HTTP сервер.")
	urlPrefixFlag := flag.String("b", "http://localhost:8080/", "Префикс для коротких URL.")
	fileFlag := flag.String("f", "urls.json", "Файл для сохранения URL.")
	flag.Parse()

	host := getConfigValue("SERVER_ADDRESS", hostFlag)
	urlPrefix := getConfigValue("BASE_URL", urlPrefixFlag)
	filePath := getConfigValue("FILE_STORAGE_PATH", fileFlag)

	return &AppConfig{
		Addr:     host,
		BaseURL:  urlPrefix,
		FilePath: filePath,
	}
}
