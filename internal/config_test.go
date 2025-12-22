package internal

import (
	"flag"
	"os"
	"testing"
)

// Тест для парсера конфигурации приложения с отсутствием переопределений.
func TestParseAppConfig(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd"}

	cfg := ParseAppConfig()
	if cfg.Host != "localhost:8080" {
		t.Fatalf("expected default host, got %q", cfg.Host)
	}

	if cfg.URLPrefix != "http://localhost:8080/" {
		t.Fatalf("expected default URLPrefix, got %q", cfg.URLPrefix)
	}
}

// Тест для парсера конфигурации приложения с переопределением через переменные окружения и флаги.
func TestParseAppConfig_Override(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "127.0.0.1:8080")

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd", "-a", "0.0.0.0:9000"}

	cfg := ParseAppConfig()
	if cfg.Host != "127.0.0.1:8080" {
		t.Fatalf("expected env override, got %q", cfg.Host)
	}
}
