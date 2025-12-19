package main

import (
	"github.com/Pelfox/go-shortener/internal"
	"github.com/Pelfox/go-shortener/internal/service"
	"github.com/rs/zerolog/log"
)

func main() {
	appConfig := internal.ParseAppConfig()

	server := service.NewServer(appConfig)
	if err := server.ServeHTTP(); err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}
}
