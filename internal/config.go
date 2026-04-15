package internal

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/rs/zerolog"
)

// generate:reset
// AppConfig хранит конфигурацию приложения.
type AppConfig struct {
	// Addr это адрес для HTTP-сервера.
	Addr string `json:"server_address"`
	// BaseURL это базовый URL для коротких ссылок.
	BaseURL string `json:"base_url"`
	// FilePath это путь к файлу для хранения ссылок.
	FilePath string `json:"file_path"`
	// DatabaseDSN это строка подключения к базе данных.
	DatabaseDSN string `json:"database_dsn"`
	// Secret это секрет для HMAC генерации пользовательского ID.
	Secret []byte `json:"secret"`
	// AuditFile это путь к файлу для сохранения аудит-событий.
	AuditFile string `json:"audit_file"`
	// AuditURL это полный URL до сервера аудит-событий.
	AuditURL string `json:"audit_url"`
	// EnableHTTPS включает HTTPS сервер.
	EnableHTTPS bool `json:"enable_https"`
}

func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// ParseAppConfig парсит конфигурацию приложения из флагов командной строки,
// переменных окружения и опционального JSON файла.
func ParseAppConfig(logger zerolog.Logger) *AppConfig {
	var configFilePath string
	flag.StringVar(&configFilePath, "config", "", "Путь до JSON-файла с конфигурацией")

	hostFlag := flag.String("a", "localhost:8080", "Адрес, на котором будет запущен HTTP сервер.")
	urlPrefixFlag := flag.String("b", "http://localhost:8080/", "Префикс для коротких URL.")
	fileFlag := flag.String("f", "urls.json", "Файл для сохранения URL.")
	databaseDSNFlag := flag.String("d", "", "Строка подключения к базе данных. Пустое значение отключает БД.")
	secretFlag := flag.String("secret", "", "Секрет для HMAC.")
	enableHTTPSFlag := flag.Bool("s", false, "Включить HTTPS.")
	auditFileFlag := flag.String("audit-file", "", "Путь к файлу для сохранения аудит событий.")
	auditURLFlag := flag.String("audit-url", "", "Полный URL для аудит сервера.")

	flag.Parse()
	if configPath, ok := os.LookupEnv("CONFIG"); ok {
		configFilePath = configPath
	}

	var config AppConfig
	if configFilePath != "" {
		file, err := os.Open(configFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Error().Msg("configuration file at the given path doesn't exist")
			} else {
				logger.Error().Err(err).Msg("failed to open configuration file")
			}
		} else {
			decoder := json.NewDecoder(file)
			if err := decoder.Decode(&config); err != nil {
				logger.Error().Err(err).Msg("failed to decode the configuration")
			}
			file.Close()
		}
	}

	// Сначала проверяем все флаги
	if isFlagSet("a") {
		config.Addr = *hostFlag
	}
	if isFlagSet("b") {
		config.BaseURL = *urlPrefixFlag
	}
	if isFlagSet("f") {
		config.FilePath = *fileFlag
	}
	if isFlagSet("d") {
		config.DatabaseDSN = *databaseDSNFlag
	}
	if isFlagSet("secret") {
		config.Secret = []byte(*secretFlag)
	}
	if isFlagSet("s") {
		config.EnableHTTPS = *enableHTTPSFlag
	}
	if isFlagSet("audit-file") {
		config.AuditFile = *auditFileFlag
	}
	if isFlagSet("audit-url") {
		config.AuditURL = *auditURLFlag
	}

	// Проверяем все переменные окружения
	if envValue, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		config.Addr = envValue
	}
	if envValue, ok := os.LookupEnv("BASE_URL"); ok {
		config.BaseURL = envValue
	}
	if envValue, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		config.FilePath = envValue
	}
	if envValue, ok := os.LookupEnv("DATABASE_DSN"); ok {
		config.DatabaseDSN = envValue
	}
	if envValue, ok := os.LookupEnv("SECRET"); ok {
		config.Secret = []byte(envValue)
	}
	if envValue, ok := os.LookupEnv("AUDIT_FILE"); ok {
		config.AuditFile = envValue
	}
	if envValue, ok := os.LookupEnv("AUDIT_URL"); ok {
		config.AuditURL = envValue
	}
	if envValue, ok := os.LookupEnv("ENABLE_HTTPS"); ok {
		config.EnableHTTPS = envValue == "true" || envValue == "1" || envValue == "t"
	}

	return &config
}
