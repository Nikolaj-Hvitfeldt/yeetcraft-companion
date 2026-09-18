package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

const (
	SettingInstallationID = "installation_id"
	SettingWoWLogTimezone = "wow_log_timezone"

	defaultBusyTimeout = 5000
)

// DB wraps a local SQLite database for companion capture state.
type DB struct {
	sql *sql.DB
}

// Open opens or creates the database at path, applies migrations, and configures pragmas.
func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := configurePragmas(ctx, sqlDB); err != nil {
		sqlDB.Close()
		return nil, err
	}
	if err := migrate(ctx, sqlDB); err != nil {
		sqlDB.Close()
		return nil, err
	}

	return &DB{sql: sqlDB}, nil
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	if db == nil || db.sql == nil {
		return nil
	}
	return db.sql.Close()
}

// SQL exposes the underlying database handle for tests that need transaction control.
func (db *DB) SQL() *sql.DB {
	return db.sql
}

func configurePragmas(ctx context.Context, db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		fmt.Sprintf("PRAGMA busy_timeout=%d", defaultBusyTimeout),
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("configure %s: %w", pragma, err)
		}
	}
	return nil
}

// GetSetting returns a settings value or empty string when unset.
func (db *DB) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := db.sql.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, nil
}

// SetSetting persists a settings value.
func (db *DB) SetSetting(ctx context.Context, key, value string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := db.sql.ExecContext(ctx, `
		INSERT INTO settings(key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at
	`, key, value, now)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}
