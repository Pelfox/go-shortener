package internal

import (
	"flag"
	"os"
	"testing"

	"github.com/rs/zerolog"
)

// Тест для парсера конфигурации приложения с отсутствием переопределений.
func TestParseAppConfig(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd"}

	cfg, err := ParseAppConfig(zerolog.New(nil))
	if err != nil {
		t.Fatalf("unexpected configuration error: %v", err)
	}

	if cfg.Addr != "localhost:8080" {
		t.Fatalf("expected default host, got %q", cfg.Addr)
	}

	if cfg.GRPCAddr != "localhost:3200" {
		t.Fatalf("expected default gRPC host, got %q", cfg.GRPCAddr)
	}

	if cfg.BaseURL != "http://localhost:8080/" {
		t.Fatalf("expected default BaseURL, got %q", cfg.BaseURL)
	}
}

// Тест для парсера конфигурации приложения с переопределением через переменные окружения и флаги.
func TestParseAppConfig_Override(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "127.0.0.1:8080")
	t.Setenv("GRPC_SERVER_ADDRESS", "127.0.0.1:3200")
	t.Setenv("ENABLE_GRPC_TLS", "true")

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd", "-a", "0.0.0.0:9000", "-g", "0.0.0.0:3000"}

	cfg, err := ParseAppConfig(zerolog.New(nil))
	if err != nil {
		t.Fatalf("unexpected configuration error: %v", err)
	}

	if cfg.Addr != "127.0.0.1:8080" {
		t.Fatalf("expected env override, got %q", cfg.Addr)
	}

	if cfg.GRPCAddr != "127.0.0.1:3200" {
		t.Fatalf("expected gRPC env override, got %q", cfg.GRPCAddr)
	}

	if !cfg.EnableGRPCTLS {
		t.Fatal("expected gRPC TLS to be enabled from env")
	}
}
