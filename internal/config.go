package internal

import (
	"flag"
	"os"
)

type AppConfig struct {
	Addr     string // адрес для старта HTTP сервера
	BaseURL  string // префикс для коротких URL
	FilePath string // путь до файла сохранения
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
