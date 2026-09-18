package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	ReviewStatusPending = "pending"
	ReviewStatusHeld    = "held"
)

// CauseInput is privacy-safe ranked cause evidence for persistence.
type CauseInput struct {
	Rank              int
	SourceType        string
	SpellID           int64
	CreatureID        int64
	HasCreatureID     bool
	EnvironmentalType string
	Amount            int64
	Overkill          int64
	PlayerOrigin      bool
	Confidence        string
}

// EventInput describes a death event to persist inside a commit transaction.
type EventInput struct {
	ClientEventID string
	CharacterGUID string
	DeathInstant  string
	Ordinal       int
	HoldReason    string
	ReviewStatus  string
	Category      string
	Confidence    string
	PayloadJSON   string
	Causes        []CauseInput
}

// EventRecord is a persisted death event with its deterministic client_event_id.
type EventRecord struct {
	ID            int64
	ClientEventID string
	RunID         int64
	CharacterGUID string
	DeathInstant  string
	Ordinal       int
	HoldReason    string
	ReviewStatus  string
	Category      string
	Confidence    string
	PayloadJSON   string
	CreatedAt     time.Time
	Causes        []CauseRecord
}

// CauseRecord is a persisted ranked cause for an event.
type CauseRecord struct {
	Rank              int
	SourceType        string
	SpellID           int64
	CreatureID        int64
	HasCreatureID     bool
	EnvironmentalType string
	Amount            int64
	Overkill          int64
	PlayerOrigin      bool
	Confidence        string
}

// CountEvents returns the number of persisted events.
func (db *DB) CountEvents(ctx context.Context) (int, error) {
	var count int
	err := db.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}
	return count, nil
}

// GetEventByClientEventID returns an event by its deterministic client_event_id.
func (db *DB) GetEventByClientEventID(ctx context.Context, clientEventID string) (*EventRecord, error) {
	row := db.sql.QueryRowContext(ctx, `
		SELECT id, client_event_id, run_id, character_guid, death_instant, ordinal,
		       hold_reason, review_status, category, confidence, payload_json, created_at
		FROM events
		WHERE client_event_id = ?
	`, clientEventID)
	rec, err := scanEvent(row)
	if err != nil || rec == nil {
		return rec, err
	}
	rec.Causes, err = loadCauses(ctx, db.sql, rec.ID)
	return rec, err
}

// ListEventsByRunID returns all events for a run ordered by death instant and ordinal.
func (db *DB) ListEventsByRunID(ctx context.Context, runID int64) ([]EventRecord, error) {
	rows, err := db.sql.QueryContext(ctx, `
		SELECT id, client_event_id, run_id, character_guid, death_instant, ordinal,
		       hold_reason, review_status, category, confidence, payload_json, created_at
		FROM events
		WHERE run_id = ?
		ORDER BY death_instant, ordinal
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var out []EventRecord
	for rows.Next() {
		rec, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		rec.Causes, err = loadCauses(ctx, db.sql, rec.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return out, nil
}

func insertEventTx(ctx context.Context, tx *sql.Tx, runID int64, input EventInput) (int64, bool, error) {
	if input.ReviewStatus == "" {
		if input.HoldReason != "" {
			input.ReviewStatus = ReviewStatusHeld
		} else {
			input.ReviewStatus = ReviewStatusPending
		}
	}
	if input.PayloadJSON == "" {
		input.PayloadJSON = "{}"
	}
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	result, err := tx.ExecContext(ctx, `
		INSERT INTO events(
			client_event_id, run_id, character_guid, death_instant, ordinal,
			hold_reason, review_status, category, confidence, payload_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_event_id) DO NOTHING
	`,
		input.ClientEventID,
		runID,
		input.CharacterGUID,
		input.DeathInstant,
		input.Ordinal,
		input.HoldReason,
		input.ReviewStatus,
		input.Category,
		input.Confidence,
		input.PayloadJSON,
		createdAt,
	)
	if err != nil {
		return 0, false, fmt.Errorf("insert event: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, false, fmt.Errorf("rows affected: %w", err)
	}
	inserted := affected > 0
	var eventID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM events WHERE client_event_id = ?`, input.ClientEventID).Scan(&eventID)
	if err != nil {
		return 0, false, fmt.Errorf("lookup event id: %w", err)
	}
	if !inserted {
		return eventID, false, nil
	}

	for _, cause := range input.Causes {
		if err := insertCauseTx(ctx, tx, eventID, cause); err != nil {
			return 0, false, err
		}
	}
	return eventID, true, nil
}

func insertCauseTx(ctx context.Context, tx *sql.Tx, eventID int64, cause CauseInput) error {
	var spellID sql.NullInt64
	if cause.SpellID != 0 {
		spellID = sql.NullInt64{Int64: cause.SpellID, Valid: true}
	}
	var creatureID sql.NullInt64
	if cause.HasCreatureID {
		creatureID = sql.NullInt64{Int64: cause.CreatureID, Valid: true}
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO event_causes(
			event_id, rank, source_type, spell_id, creature_id, environmental_type,
			amount, overkill, player_origin, confidence
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		eventID,
		cause.Rank,
		cause.SourceType,
		spellID,
		creatureID,
		cause.EnvironmentalType,
		cause.Amount,
		cause.Overkill,
		boolToInt(cause.PlayerOrigin),
		cause.Confidence,
	)
	if err != nil {
		return fmt.Errorf("insert cause rank %d: %w", cause.Rank, err)
	}
	return nil
}

func loadCauses(ctx context.Context, q queryer, eventID int64) ([]CauseRecord, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT rank, source_type, spell_id, creature_id, environmental_type,
		       amount, overkill, player_origin, confidence
		FROM event_causes
		WHERE event_id = ?
		ORDER BY rank
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("load causes: %w", err)
	}
	defer rows.Close()

	var out []CauseRecord
	for rows.Next() {
		var rec CauseRecord
		var spellID sql.NullInt64
		var creatureID sql.NullInt64
		var playerOrigin int
		if err := rows.Scan(
			&rec.Rank,
			&rec.SourceType,
			&spellID,
			&creatureID,
			&rec.EnvironmentalType,
			&rec.Amount,
			&rec.Overkill,
			&playerOrigin,
			&rec.Confidence,
		); err != nil {
			return nil, fmt.Errorf("scan cause: %w", err)
		}
		if spellID.Valid {
			rec.SpellID = spellID.Int64
		}
		if creatureID.Valid {
			rec.CreatureID = creatureID.Int64
			rec.HasCreatureID = true
		}
		rec.PlayerOrigin = playerOrigin != 0
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate causes: %w", err)
	}
	return out, nil
}

func scanEvent(row interface {
	Scan(dest ...any) error
}) (*EventRecord, error) {
	var rec EventRecord
	var createdAt string
	if err := row.Scan(
		&rec.ID,
		&rec.ClientEventID,
		&rec.RunID,
		&rec.CharacterGUID,
		&rec.DeathInstant,
		&rec.Ordinal,
		&rec.HoldReason,
		&rec.ReviewStatus,
		&rec.Category,
		&rec.Confidence,
		&rec.PayloadJSON,
		&createdAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan event: %w", err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse event created_at: %w", err)
	}
	rec.CreatedAt = parsed
	return &rec, nil
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
