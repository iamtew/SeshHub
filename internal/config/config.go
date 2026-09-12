package config

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	AppEnv              string
	Port                string
	BaseURL             string
	SessionSecret       string
	DatabaseURL         string
	WebDir              string
	MigrationsDir       string
	DiscordClientID     string
	DiscordClientSecret string
	DiscordGuildID      string
	DiscordAdminRoleID  string
	DiscordSkaterRoleID string
	SuperAdminIDs       []string
	YouTubeClientID     string
	YouTubeClientSecret string
}

func Load() Config {
	loadDotEnv(".env")
	return Config{
		AppEnv:              getenv("APP_ENV", "development"),
		Port:                getenv("PORT", "53053"),
		BaseURL:             getenv("BASE_URL", "http://localhost:53053"),
		SessionSecret:       getenv("SESSION_SECRET", "change-me-to-a-secure-random-32-byte-hex-string"),
		DatabaseURL:         getenv("DATABASE_URL", "file:seshhub.db"),
		WebDir:              getenv("WEB_DIR", "web"),
		MigrationsDir:       getenv("MIGRATIONS_DIR", filepath.Join("internal", "db", "migrations")),
		DiscordClientID:     getenv("DISCORD_CLIENT_ID", ""),
		DiscordClientSecret: getenv("DISCORD_CLIENT_SECRET", ""),
		DiscordGuildID:      getenv("DISCORD_GUILD_ID", ""),
		DiscordAdminRoleID:  getenv("DISCORD_ADMIN_ROLE_ID", ""),
		DiscordSkaterRoleID: getenv("DISCORD_SKATER_ROLE_ID", ""),
		SuperAdminIDs:       splitCSV(getenv("SUPERADMIN_DISCORD_IDS", "")),
		YouTubeClientID:     getenv("YOUTUBE_CLIENT_ID", ""),
		YouTubeClientSecret: getenv("YOUTUBE_CLIENT_SECRET", ""),
	}
}

func (c Config) DiscordEnabled() bool {
	return c.DiscordClientID != "" && c.DiscordClientSecret != ""
}

func (c Config) YouTubeEnabled() bool {
	return c.YouTubeClientID != "" && c.YouTubeClientSecret != ""
}

func (c Config) CookieSecure() bool {
	return c.AppEnv == "production"
}

// ListenAddr is host:port. -port accepts "53054" or "127.0.0.1:53054".
func (c Config) ListenAddr() string {
	p := strings.TrimSpace(c.Port)
	if p == "" {
		p = "53053"
	}
	if strings.Contains(p, ":") {
		return p
	}
	return ":" + p
}

func (c *Config) ApplyFlags(fs *flag.FlagSet, args []string) error {
	port := fs.String("port", "", "HTTP listen port or address (overrides PORT)")
	dbURL := fs.String("db", "", "database URL (overrides DATABASE_URL)")
	webDir := fs.String("web", "", "web assets directory (overrides WEB_DIR)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *port != "" {
		c.Port = *port
	}
	if *dbURL != "" {
		c.DatabaseURL = *dbURL
	}
	if *webDir != "" {
		c.WebDir = *webDir
	}
	return nil
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
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
