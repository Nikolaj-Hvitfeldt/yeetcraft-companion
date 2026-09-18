package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// FileState is the persisted read position for a combat-log file generation.
type FileState struct {
	Path            string
	FileIdentity    string
	Generation      int64
	ByteOffset      int64
	PartialLine     string
	ParserStateJSON string
}

// CommitInput atomically persists a run, events, causes, and the file offset derived from those bytes.
type CommitInput struct {
	Run    RunInput
	Events []EventInput
	File   FileState
}

// Commit persists runs, events, causes, and the file offset in a single transaction.
// Offset advancement and event persistence succeed or fail together.
func (db *DB) Commit(ctx context.Context, input CommitInput) error {
	tx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin commit transaction: %w", err)
	}
	if err := commitTx(ctx, tx, input); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("%w (rollback failed: %v)", err, rbErr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// CommitUncommitted writes input inside an open transaction without committing.
// It exists for crash-before-ack tests that need to roll back a partial write.
func (db *DB) CommitUncommitted(ctx context.Context, tx *sql.Tx, input CommitInput) error {
	return commitTx(ctx, tx, input)
}

// BeginTx starts a database transaction.
func (db *DB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return db.sql.BeginTx(ctx, nil)
}

// GetFileState returns the latest committed file state for path and identity.
func (db *DB) GetFileState(ctx context.Context, path, fileIdentity string) (*FileState, error) {
	row := db.sql.QueryRowContext(ctx, `
		SELECT path, file_identity, generation, byte_offset, partial_line, parser_state_json
		FROM files
		WHERE path = ? AND file_identity = ?
		ORDER BY generation DESC
		LIMIT 1
	`, path, fileIdentity)

	var state FileState
	if err := row.Scan(
		&state.Path,
		&state.FileIdentity,
		&state.Generation,
		&state.ByteOffset,
		&state.PartialLine,
		&state.ParserStateJSON,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get file state: %w", err)
	}
	return &state, nil
}

func commitTx(ctx context.Context, tx *sql.Tx, input CommitInput) error {
	if len(input.Events) > 0 && input.Run.ClientRunID == "" {
		return fmt.Errorf("commit events: run client_run_id required")
	}

	var runID int64
	if input.Run.ClientRunID != "" {
		var err error
		runID, err = upsertRunTx(ctx, tx, input.Run)
		if err != nil {
			return err
		}
	}

	for _, event := range input.Events {
		_, _, err := insertEventTx(ctx, tx, runID, event)
		if err != nil {
			return err
		}
	}

	if err := upsertFileStateTx(ctx, tx, input.File); err != nil {
		return err
	}
	return nil
}

func upsertFileStateTx(ctx context.Context, tx *sql.Tx, state FileState) error {
	if state.ParserStateJSON == "" {
		state.ParserStateJSON = "{}"
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO files(
			path, file_identity, generation, byte_offset, partial_line, parser_state_json, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path, file_identity, generation) DO UPDATE SET
			byte_offset = excluded.byte_offset,
			partial_line = excluded.partial_line,
			parser_state_json = excluded.parser_state_json,
			updated_at = excluded.updated_at
	`,
		state.Path,
		state.FileIdentity,
		state.Generation,
		state.ByteOffset,
		state.PartialLine,
		state.ParserStateJSON,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert file state: %w", err)
	}
	return nil
}
