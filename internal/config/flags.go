package config

import "flag"

type AppConfig struct {
	Host      string // хост для старта HTTP сервера
	URLPrefix string // префикс для коротких URL
}

func ParseAppConfig() *AppConfig {
	host := flag.String("a", "localhost:8888", "Адрес, на котором будет запущен HTTP сервер.")
	urlPrefix := flag.String("b", "http://localhost:8888/", "Префикс для коротких URL.")

	flag.Parse()
	return &AppConfig{
		Host:      *host,
		URLPrefix: *urlPrefix,
	}
}
