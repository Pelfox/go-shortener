package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/rs/zerolog"
)

// AppConfig хранит конфигурацию приложения.
type AppConfig struct {
	// Addr это адрес для HTTP-сервера.
	Addr string `json:"server_address"`
	// GRPCAddr это адрес для gRPC-сервера.
	GRPCAddr string `json:"grpc_server_address"`
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
	// EnableGRPCTLS включает TLS для gRPC-сервера.
	EnableGRPCTLS bool `json:"enable_grpc_tls"`
	// TrustedSubnet содержит в себе подсеть, которой доступен эндпоинт статистики.
	TrustedSubnet *net.IPNet
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
func ParseAppConfig(logger zerolog.Logger) (*AppConfig, error) {
	var configFilePath string
	flag.StringVar(&configFilePath, "config", "", "Путь до JSON-файла с конфигурацией")

	hostFlag := flag.String("a", "localhost:8080", "Адрес, на котором будет запущен HTTP сервер.")
	grpcHostFlag := flag.String("g", "localhost:3200", "Адрес, на котором будет запущен gRPC сервер.")
	urlPrefixFlag := flag.String("b", "http://localhost:8080/", "Префикс для коротких URL.")
	fileFlag := flag.String("f", "urls.json", "Файл для сохранения URL.")
	databaseDSNFlag := flag.String("d", "", "Строка подключения к базе данных. Пустое значение отключает БД.")
	secretFlag := flag.String("secret", "", "Секрет для HMAC.")
	enableHTTPSFlag := flag.Bool("s", false, "Включить HTTPS.")
	enableGRPCTLSFlag := flag.Bool("grpc-tls", false, "Включить TLS для gRPC.")
	auditFileFlag := flag.String("audit-file", "", "Путь к файлу для сохранения аудит событий.")
	auditURLFlag := flag.String("audit-url", "", "Полный URL для аудит сервера.")
	trustedSubnetFlag := flag.String("t", "", "Подсеть для эндпоинта статистики.")

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
	if isFlagSet("g") {
		config.GRPCAddr = *grpcHostFlag
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
	if isFlagSet("grpc-tls") {
		config.EnableGRPCTLS = *enableGRPCTLSFlag
	}
	if isFlagSet("audit-file") {
		config.AuditFile = *auditFileFlag
	}
	if isFlagSet("audit-url") {
		config.AuditURL = *auditURLFlag
	}
	if isFlagSet("t") {
		_, ipNet, err := net.ParseCIDR(*trustedSubnetFlag)
		if err != nil {
			return nil, fmt.Errorf("failed to parse trusted subnet: %w", err)
		}
		config.TrustedSubnet = ipNet
	}

	// Проверяем все переменные окружения
	if envValue, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		config.Addr = envValue
	}
	if envValue, ok := os.LookupEnv("GRPC_SERVER_ADDRESS"); ok {
		config.GRPCAddr = envValue
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
	if envValue, ok := os.LookupEnv("ENABLE_GRPC_TLS"); ok {
		config.EnableGRPCTLS = envValue == "true" || envValue == "1" || envValue == "t"
	}
	if envValue, ok := os.LookupEnv("TRUSTED_SUBNET"); ok {
		_, ipNet, err := net.ParseCIDR(envValue)
		if err != nil {
			return nil, fmt.Errorf("failed to parse trusted subnet: %w", err)
		}
		config.TrustedSubnet = ipNet
	}

	if config.Addr == "" {
		config.Addr = *hostFlag
	}
	if config.GRPCAddr == "" {
		config.GRPCAddr = *grpcHostFlag
	}
	if config.BaseURL == "" {
		config.BaseURL = *urlPrefixFlag
	}
	if config.FilePath == "" {
		config.FilePath = *fileFlag
	}

	return &config, nil
}
