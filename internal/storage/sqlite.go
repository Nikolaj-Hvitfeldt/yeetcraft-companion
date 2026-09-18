package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
	sqlDB, err := sql.Open("sqlite", pragmaDSN(path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

// pragmaDSN carries the connection pragmas in the DSN. foreign_keys and
// busy_timeout are per-connection settings, so executing them once against the
// pool would leave every later connection database/sql opens without them.
func pragmaDSN(path string) string {
	pragmas := []string{
		"journal_mode(WAL)",
		"foreign_keys(1)",
		fmt.Sprintf("busy_timeout(%d)", defaultBusyTimeout),
	}
	return path + "?_pragma=" + strings.Join(pragmas, "&_pragma=")
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
