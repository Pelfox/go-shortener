package main

import (
	"os"

	"github.com/Pelfox/go-shortener/internal"
	"github.com/Pelfox/go-shortener/internal/service"
	"github.com/rs/zerolog"
)

func main() {
	serverLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().
		Timestamp().
		Str("component", "server").
		Logger()
	middlewareLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().
		Timestamp().
		Str("component", "logger-middleware").
		Logger()

	appConfig := internal.ParseAppConfig()

	storage := internal.NewInMemoryStorage(appConfig.FilePath)
	server := service.NewServer(appConfig, serverLogger, middlewareLogger, storage)

	if err := server.ServeHTTP(); err != nil {
		serverLogger.Fatal().Err(err).Msg("failed to start server")
	}
}
