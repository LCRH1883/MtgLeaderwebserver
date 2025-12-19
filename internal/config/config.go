package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	Env          string
	Addr         string
	PublicURL    *url.URL
	DBDSN        string
	CookieSecret string
	SessionTTL   time.Duration
	LogLevel     string
	AdminEmails  []string

	AdminBootstrapEmail    string
	AdminBootstrapUsername string
	AdminBootstrapPassword string
}

func Load() (Config, error) {
	return LoadFromEnv(os.Getenv)
}

func LoadFromEnv(getenv func(string) string) (Config, error) {
	cfg := Config{
		Env:          getenv("APP_ENV"),
		Addr:         getenv("APP_ADDR"),
		DBDSN:        getenv("APP_DB_DSN"),
		LogLevel:     getenv("APP_LOG_LEVEL"),
		CookieSecret: getenv("APP_COOKIE_SECRET"),
	}

	if cfg.Env == "" {
		cfg.Env = "dev"
	}
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:8080"
	}

	publicURLRaw := getenv("APP_PUBLIC_URL")
	if publicURLRaw != "" {
		parsed, err := url.Parse(publicURLRaw)
		if err != nil {
			return Config{}, fmt.Errorf("APP_PUBLIC_URL: %w", err)
		}
		if !parsed.IsAbs() || parsed.Host == "" {
			return Config{}, errors.New("APP_PUBLIC_URL: must be an absolute URL")
		}
		switch parsed.Scheme {
		case "http", "https":
		default:
			return Config{}, errors.New("APP_PUBLIC_URL: scheme must be http or https")
		}
		cfg.PublicURL = parsed
	}

	ttlRaw := getenv("APP_SESSION_TTL")
	if ttlRaw == "" {
		cfg.SessionTTL = 30 * 24 * time.Hour
	} else {
		ttl, err := time.ParseDuration(ttlRaw)
		if err != nil {
			return Config{}, fmt.Errorf("APP_SESSION_TTL: %w", err)
		}
		if ttl <= 0 {
			return Config{}, errors.New("APP_SESSION_TTL: must be > 0")
		}
		cfg.SessionTTL = ttl
	}

	switch cfg.Env {
	case "dev", "prod", "test":
	default:
		return Config{}, errors.New("APP_ENV: must be one of dev, test, prod")
	}

	cfg.AdminEmails = parseCSV(getenv("APP_ADMIN_EMAILS"))
	cfg.AdminBootstrapEmail = strings.TrimSpace(strings.ToLower(getenv("APP_ADMIN_BOOTSTRAP_EMAIL")))
	cfg.AdminBootstrapUsername = strings.TrimSpace(getenv("APP_ADMIN_BOOTSTRAP_USERNAME"))
	cfg.AdminBootstrapPassword = getenv("APP_ADMIN_BOOTSTRAP_PASSWORD")

	if cfg.AdminBootstrapPassword != "" && cfg.AdminBootstrapEmail == "" {
		return Config{}, errors.New("APP_ADMIN_BOOTSTRAP_EMAIL: required when APP_ADMIN_BOOTSTRAP_PASSWORD is set")
	}
	if cfg.AdminBootstrapPassword != "" && cfg.AdminBootstrapUsername == "" {
		cfg.AdminBootstrapUsername = "admin"
	}
	if cfg.AdminBootstrapEmail != "" && !contains(cfg.AdminEmails, cfg.AdminBootstrapEmail) {
		cfg.AdminEmails = append(cfg.AdminEmails, cfg.AdminBootstrapEmail)
	}

	if cfg.IsProd() {
		if cfg.PublicURL == nil {
			return Config{}, errors.New("APP_PUBLIC_URL: required in prod")
		}
		if cfg.DBDSN == "" {
			return Config{}, errors.New("APP_DB_DSN: required in prod")
		}
		if len(cfg.CookieSecret) < 32 {
			return Config{}, errors.New("APP_COOKIE_SECRET: must be at least 32 bytes in prod")
		}
	}

	return cfg, nil
}

func (c Config) IsProd() bool { return c.Env == "prod" }

func (c Config) CookieSecure() bool {
	if c.PublicURL != nil {
		return c.PublicURL.Scheme == "https"
	}
	return c.IsProd()
}

func parseCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func contains(ss []string, needle string) bool {
	for _, s := range ss {
		if s == needle {
			return true
		}
	}
	return false
}
