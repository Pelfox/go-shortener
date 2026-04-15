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

	cfg := ParseAppConfig(zerolog.New(nil))
	if cfg.Addr != "localhost:8080" {
		t.Fatalf("expected default host, got %q", cfg.Addr)
	}

	if cfg.BaseURL != "http://localhost:8080/" {
		t.Fatalf("expected default BaseURL, got %q", cfg.BaseURL)
	}
}

// Тест для парсера конфигурации приложения с переопределением через переменные окружения и флаги.
func TestParseAppConfig_Override(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "127.0.0.1:8080")

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd", "-a", "0.0.0.0:9000"}

	cfg := ParseAppConfig(zerolog.New(nil))
	if cfg.Addr != "127.0.0.1:8080" {
		t.Fatalf("expected env override, got %q", cfg.Addr)
	}
}
