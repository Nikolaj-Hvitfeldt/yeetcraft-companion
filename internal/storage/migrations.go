package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const currentSchemaVersion = 1

var migrationStatements = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY,
		path TEXT NOT NULL,
		file_identity TEXT NOT NULL,
		generation INTEGER NOT NULL DEFAULT 0,
		byte_offset INTEGER NOT NULL DEFAULT 0,
		partial_line TEXT NOT NULL DEFAULT '',
		parser_state_json TEXT NOT NULL DEFAULT '{}',
		updated_at TEXT NOT NULL,
		UNIQUE(path, file_identity, generation)
	)`,
	`CREATE TABLE IF NOT EXISTS runs (
		id INTEGER PRIMARY KEY,
		client_run_id TEXT NOT NULL UNIQUE,
		challenge_mode_start_instant TEXT NOT NULL,
		challenge_map_id INTEGER NOT NULL,
		keystone_level INTEGER NOT NULL,
		status TEXT NOT NULL,
		metadata_json TEXT NOT NULL DEFAULT '{}',
		started_at TEXT NOT NULL,
		ended_at TEXT
	)`,
	`CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY,
		client_event_id TEXT NOT NULL UNIQUE,
		run_id INTEGER NOT NULL REFERENCES runs(id),
		character_guid TEXT NOT NULL,
		death_instant TEXT NOT NULL,
		ordinal INTEGER NOT NULL,
		hold_reason TEXT NOT NULL DEFAULT '',
		review_status TEXT NOT NULL DEFAULT 'pending',
		category TEXT NOT NULL DEFAULT '',
		confidence TEXT NOT NULL DEFAULT '',
		payload_json TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL,
		UNIQUE(run_id, character_guid, death_instant, ordinal)
	)`,
	`CREATE TABLE IF NOT EXISTS event_causes (
		id INTEGER PRIMARY KEY,
		event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
		rank INTEGER NOT NULL,
		source_type TEXT NOT NULL,
		spell_id INTEGER,
		creature_id INTEGER,
		environmental_type TEXT NOT NULL DEFAULT '',
		amount INTEGER NOT NULL DEFAULT 0,
		overkill INTEGER NOT NULL DEFAULT 0,
		player_origin INTEGER NOT NULL DEFAULT 0,
		confidence TEXT NOT NULL,
		UNIQUE(event_id, rank)
	)`,
	`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
}

func migrate(ctx context.Context, db *sql.DB) error {
	for _, stmt := range migrationStatements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply migration statement: %w", err)
		}
	}

	var version int
	err := db.QueryRowContext(ctx, `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version)
	if err == sql.ErrNoRows {
		_, err = db.ExecContext(ctx,
			`INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`,
			currentSchemaVersion,
			time.Now().UTC().Format(time.RFC3339Nano),
		)
		if err != nil {
			return fmt.Errorf("record schema version: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version > currentSchemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported %d", version, currentSchemaVersion)
	}
	if version < currentSchemaVersion {
		return fmt.Errorf("unsupported schema upgrade from %d to %d", version, currentSchemaVersion)
	}
	return nil
}
