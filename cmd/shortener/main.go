package main

import (
	"github.com/Pelfox/go-shortener/internal"
	"github.com/Pelfox/go-shortener/internal/service"
)

func main() {
	appConfig := internal.ParseAppConfig()

	server := service.NewServer(appConfig.Host, appConfig.URLPrefix)
	if err := server.ServeHTTP(); err != nil {
		panic(err)
	}
}
