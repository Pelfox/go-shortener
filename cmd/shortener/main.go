package main

import "github.com/Pelfox/go-shortener/internal/service"

func main() {
	server := service.NewServer("localhost:8080")
	if err := server.ServeHTTP(); err != nil {
		panic(err)
	}
}
