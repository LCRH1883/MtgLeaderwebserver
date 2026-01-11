package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	err := os.WriteFile(path, []byte(`# comment
APP_ADDR=127.0.0.1:8081
export APP_DB_DSN="postgres://user:pass@127.0.0.1:5432/mtg?sslmode=disable"
APP_COOKIE_SECRET='supersecret'
INVALID_LINE
EMPTY=
`), 0o600)
	if err != nil {
		t.Fatalf("write env file: %v", err)
	}

	env := map[string]string{
		"APP_ADDR": "127.0.0.1:8080",
	}
	getenv := func(k string) string { return env[k] }
	setenv := func(k, v string) error {
		env[k] = v
		return nil
	}

	if err := loadDotEnvFile(path, setenv, getenv); err != nil {
		t.Fatalf("loadDotEnvFile: %v", err)
	}

	if got := env["APP_ADDR"]; got != "127.0.0.1:8080" {
		t.Fatalf("APP_ADDR override: got %q", got)
	}
	if got := env["APP_DB_DSN"]; got != "postgres://user:pass@127.0.0.1:5432/mtg?sslmode=disable" {
		t.Fatalf("APP_DB_DSN: got %q", got)
	}
	if got := env["APP_COOKIE_SECRET"]; got != "supersecret" {
		t.Fatalf("APP_COOKIE_SECRET: got %q", got)
	}
	if _, ok := env["EMPTY"]; ok {
		t.Fatalf("EMPTY: expected not set, got %q", env["EMPTY"])
	}
}
