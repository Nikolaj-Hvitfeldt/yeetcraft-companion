package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// RunStatus values for local run lifecycle tracking.
const (
	RunStatusActive    = "active"
	RunStatusCompleted = "completed"
	RunStatusAbandoned = "abandoned"
)

// RunRecord is a persisted Mythic+ run with deterministic client_run_id.
type RunRecord struct {
	ID                        int64
	ClientRunID               string
	ChallengeModeStartInstant string
	ChallengeMapID            int64
	KeystoneLevel             int64
	Status                    string
	MetadataJSON              string
	StartedAt                 time.Time
	EndedAt                   *time.Time
}

// RunInput describes a run to persist inside a commit transaction.
type RunInput struct {
	ClientRunID               string
	ChallengeModeStartInstant string
	ChallengeMapID            int64
	KeystoneLevel             int64
	Status                    string
	MetadataJSON              string
	StartedAt                 time.Time
}

// GetActiveRun returns the most recently started active run, if any.
func (db *DB) GetActiveRun(ctx context.Context) (*RunRecord, error) {
	row := db.sql.QueryRowContext(ctx, `
		SELECT id, client_run_id, challenge_mode_start_instant, challenge_map_id,
		       keystone_level, status, metadata_json, started_at, ended_at
		FROM runs
		WHERE status = ?
		ORDER BY started_at DESC
		LIMIT 1
	`, RunStatusActive)
	return scanRun(row)
}

// UpdateRunStatus sets the terminal or transitional status for a persisted run.
func (db *DB) UpdateRunStatus(ctx context.Context, clientRunID, status string, endedAt time.Time) error {
	result, err := db.sql.ExecContext(ctx, `
		UPDATE runs
		SET status = ?, ended_at = ?
		WHERE client_run_id = ?
	`, status, endedAt.UTC().Format(time.RFC3339Nano), clientRunID)
	if err != nil {
		return fmt.Errorf("update run status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update run status rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("update run status: run %q not found", clientRunID)
	}
	return nil
}

// CloseRun records a terminal status for a run, inserting it first when capture
// never committed the run while it was active.
func (db *DB) CloseRun(ctx context.Context, input RunInput, status string, endedAt time.Time) error {
	tx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin close run transaction: %w", err)
	}

	input.Status = status
	if _, err := upsertRunTx(ctx, tx, input); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE runs
		SET status = ?, ended_at = ?
		WHERE client_run_id = ?
	`, status, endedAt.UTC().Format(time.RFC3339Nano), input.ClientRunID); err != nil {
		tx.Rollback()
		return fmt.Errorf("close run: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit close run: %w", err)
	}
	return nil
}

// GetRunByClientRunID returns a run by its deterministic client_run_id.
func (db *DB) GetRunByClientRunID(ctx context.Context, clientRunID string) (*RunRecord, error) {
	row := db.sql.QueryRowContext(ctx, `
		SELECT id, client_run_id, challenge_mode_start_instant, challenge_map_id,
		       keystone_level, status, metadata_json, started_at, ended_at
		FROM runs
		WHERE client_run_id = ?
	`, clientRunID)
	return scanRun(row)
}

func upsertRunTx(ctx context.Context, tx *sql.Tx, input RunInput) (int64, error) {
	if input.MetadataJSON == "" {
		input.MetadataJSON = "{}"
	}
	startedAt := input.StartedAt.UTC().Format(time.RFC3339Nano)

	_, err := tx.ExecContext(ctx, `
		INSERT INTO runs(
			client_run_id, challenge_mode_start_instant, challenge_map_id,
			keystone_level, status, metadata_json, started_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_run_id) DO NOTHING
	`,
		input.ClientRunID,
		input.ChallengeModeStartInstant,
		input.ChallengeMapID,
		input.KeystoneLevel,
		input.Status,
		input.MetadataJSON,
		startedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("upsert run: %w", err)
	}

	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM runs WHERE client_run_id = ?`, input.ClientRunID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("lookup run id: %w", err)
	}
	return id, nil
}

func scanRun(row interface {
	Scan(dest ...any) error
}) (*RunRecord, error) {
	var rec RunRecord
	var startedAt string
	var endedAt sql.NullString
	if err := row.Scan(
		&rec.ID,
		&rec.ClientRunID,
		&rec.ChallengeModeStartInstant,
		&rec.ChallengeMapID,
		&rec.KeystoneLevel,
		&rec.Status,
		&rec.MetadataJSON,
		&startedAt,
		&endedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan run: %w", err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return nil, fmt.Errorf("parse run started_at: %w", err)
	}
	rec.StartedAt = parsed
	if endedAt.Valid {
		parsedEnd, err := time.Parse(time.RFC3339Nano, endedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse run ended_at: %w", err)
		}
		rec.EndedAt = &parsedEnd
	}
	return &rec, nil
}
