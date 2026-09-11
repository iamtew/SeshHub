package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	AppEnv        string
	Port          string
	BaseURL       string
	SessionSecret string
	DatabaseURL   string
	WebDir        string
	MigrationsDir string
}

func Load() Config {
	loadDotEnv(".env")
	return Config{
		AppEnv:        getenv("APP_ENV", "development"),
		Port:          getenv("PORT", "53053"),
		BaseURL:       getenv("BASE_URL", "http://localhost:53053"),
		SessionSecret: getenv("SESSION_SECRET", "change-me-to-a-secure-random-32-byte-hex-string"),
		DatabaseURL:   getenv("DATABASE_URL", "file:seshhub.db"),
		WebDir:        getenv("WEB_DIR", "web"),
		MigrationsDir: getenv("MIGRATIONS_DIR", filepath.Join("internal", "db", "migrations")),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadDotEnv(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, strings.TrimSpace(v))
		}
	}
}
