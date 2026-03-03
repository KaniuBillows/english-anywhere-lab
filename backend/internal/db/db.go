package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bennyshi/english-anywhere-lab/internal/config"
	_ "modernc.org/sqlite"
)

func Open(cfg *config.Config) (*sql.DB, error) {
	// Ensure data directory exists
	dir := filepath.Dir(cfg.SQLitePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dsn := fmt.Sprintf("%s?_busy_timeout=%d", cfg.SQLitePath, cfg.SQLiteBusyTimeout)
	if cfg.SQLiteWAL {
		dsn += "&_journal_mode=WAL"
	}

	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Set pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		fmt.Sprintf("PRAGMA busy_timeout = %d", cfg.SQLiteBusyTimeout),
	}
	for _, p := range pragmas {
		if _, err := database.Exec(p); err != nil {
			database.Close()
			return nil, fmt.Errorf("exec pragma %q: %w", p, err)
		}
	}

	// SQLite works best with a single connection for writes
	database.SetMaxOpenConns(1)

	return database, nil
}
