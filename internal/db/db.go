package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

func Open(databaseURL string) (*sql.DB, error) {
	dsn := databaseURL
	if strings.HasPrefix(dsn, "file:") {
		dsn = strings.TrimPrefix(dsn, "file:")
	}
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	sqldb.SetMaxOpenConns(1)
	if err := sqldb.Ping(); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return sqldb, nil
}

// ponytail: rerun all CREATE IF NOT EXISTS on start; versioned migrate when a file needs ALTER.
func Migrate(sqldb *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading migrations: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}
		if _, err := sqldb.Exec(string(body)); err != nil {
			return fmt.Errorf("applying %s: %w", name, err)
		}
	}
	return ensureUserYouTubeCols(sqldb)
}

func ensureUserYouTubeCols(sqldb *sql.DB) error {
	for _, stmt := range []string{
		`ALTER TABLE users ADD COLUMN youtube_refresh_token TEXT`,
		`ALTER TABLE users ADD COLUMN youtube_synced_at DATETIME`,
	} {
		if _, err := sqldb.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("youtube user columns: %w", err)
		}
	}
	return nil
}
