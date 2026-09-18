package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/session"
)

const (
	testRunStart = "2026-02-10T18:30:00.000000000Z"
	testGUID     = "Player-0001-00000001"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "companion.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return db
}

func testRunInput() RunInput {
	return RunInput{
		ClientRunID:               "sha256:8de6d05a987d9230788f5a56957a80c8608335bbc509ca5fac1a1bf17336b30d",
		ChallengeModeStartInstant: testRunStart,
		ChallengeMapID:            375,
		KeystoneLevel:             12,
		Status:                    RunStatusActive,
		StartedAt:                 time.Date(2026, 2, 10, 18, 30, 0, 0, time.UTC),
	}
}

func testEventInput(clientEventID, deathInstant string, ordinal int) EventInput {
	return EventInput{
		ClientEventID: clientEventID,
		CharacterGUID: testGUID,
		DeathInstant:  deathInstant,
		Ordinal:       ordinal,
		Causes: []CauseInput{
			{
				Rank:       1,
				SourceType: "spell",
				SpellID:    900001,
				Amount:     50000,
				Overkill:   12000,
				Confidence: "high",
			},
		},
	}
}

func mustClientEventID(t *testing.T, deathInstant string, ordinal int) string {
	t.Helper()
	runID, err := session.ClientRunID(session.RunIDParams{
		ChallengeModeStartInstant: testRunStart,
		ChallengeMapID:            375,
		KeystoneLevel:             12,
	})
	if err != nil {
		t.Fatalf("ClientRunID: %v", err)
	}
	eventID, err := session.ClientEventID(session.EventIDParams{
		ClientRunID:   runID,
		CharacterGUID: testGUID,
		DeathInstant:  deathInstant,
		Ordinal:       ordinal,
	})
	if err != nil {
		t.Fatalf("ClientEventID: %v", err)
	}
	return eventID
}

func testFileState(offset int64) FileState {
	return FileState{
		Path:            "/tmp/WoWCombatLog.txt",
		FileIdentity:    "identity-1",
		Generation:      0,
		ByteOffset:      offset,
		PartialLine:     "",
		ParserStateJSON: `{"format":"V22"}`,
	}
}

func TestMigrationIdempotency(t *testing.T) {
	path := filepath.Join(t.TempDir(), "companion.db")

	db1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	db2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer db2.Close()
}

func TestClientEventIDUniqueness(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	eventID := mustClientEventID(t, "2026-02-10T18:45:12.123456789Z", 0)

	if err := db.Commit(ctx, CommitInput{
		Run: testRunInput(),
		Events: []EventInput{
			testEventInput(eventID, "2026-02-10T18:45:12.123456789Z", 0),
		},
		File: testFileState(100),
	}); err != nil {
		t.Fatalf("first Commit: %v", err)
	}

	tx, err := db.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer tx.Rollback()

	runID, err := upsertRunTx(ctx, tx, testRunInput())
	if err != nil {
		t.Fatalf("upsertRunTx: %v", err)
	}

	duplicate := testEventInput("sha256:0000000000000000000000000000000000000000000000000000000000000001", "2026-02-10T18:45:12.123456789Z", 0)
	_, _, err = insertEventTx(ctx, tx, runID, duplicate)
	if err == nil {
		t.Fatal("expected uniqueness violation for duplicate ordinal, got nil")
	}
}

func TestCommitWritesRunEventsCausesAndOffsetTogether(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	eventID := mustClientEventID(t, "2026-02-10T18:45:12.123456789Z", 0)
	input := CommitInput{
		Run: testRunInput(),
		Events: []EventInput{
			testEventInput(eventID, "2026-02-10T18:45:12.123456789Z", 0),
		},
		File: testFileState(256),
	}
	if err := db.Commit(ctx, input); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	run, err := db.GetRunByClientRunID(ctx, input.Run.ClientRunID)
	if err != nil {
		t.Fatalf("GetRunByClientRunID: %v", err)
	}
	if run == nil {
		t.Fatal("run not persisted")
	}

	events, err := db.ListEventsByRunID(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListEventsByRunID: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	if events[0].ClientEventID != eventID {
		t.Fatalf("client_event_id = %q, want %q", events[0].ClientEventID, eventID)
	}
	if len(events[0].Causes) != 1 {
		t.Fatalf("cause count = %d, want 1", len(events[0].Causes))
	}
	if events[0].Causes[0].SpellID != 900001 {
		t.Fatalf("spell id = %d, want 900001", events[0].Causes[0].SpellID)
	}

	state, err := db.GetFileState(ctx, input.File.Path, input.File.FileIdentity)
	if err != nil {
		t.Fatalf("GetFileState: %v", err)
	}
	if state == nil || state.ByteOffset != 256 {
		t.Fatalf("byte offset = %+v, want 256", state)
	}
	if state.ParserStateJSON != `{"format":"V22"}` {
		t.Fatalf("parser state = %q", state.ParserStateJSON)
	}
}

func TestCrashBeforeAckRollbackAndReplay(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	eventID := mustClientEventID(t, "2026-02-10T18:45:12.123456789Z", 0)
	input := CommitInput{
		Run: testRunInput(),
		Events: []EventInput{
			testEventInput(eventID, "2026-02-10T18:45:12.123456789Z", 0),
		},
		File: testFileState(512),
	}

	tx, err := db.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	if err := db.CommitUncommitted(ctx, tx, input); err != nil {
		t.Fatalf("CommitUncommitted: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	count, err := db.CountEvents(ctx)
	if err != nil {
		t.Fatalf("CountEvents after rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("event count after rollback = %d, want 0", count)
	}

	state, err := db.GetFileState(ctx, input.File.Path, input.File.FileIdentity)
	if err != nil {
		t.Fatalf("GetFileState after rollback: %v", err)
	}
	if state != nil {
		t.Fatalf("file state after rollback = %+v, want nil", state)
	}

	if err := db.Commit(ctx, input); err != nil {
		t.Fatalf("replay Commit: %v", err)
	}
	if err := db.Commit(ctx, input); err != nil {
		t.Fatalf("second replay Commit: %v", err)
	}

	count, err = db.CountEvents(ctx)
	if err != nil {
		t.Fatalf("CountEvents after replay: %v", err)
	}
	if count != 1 {
		t.Fatalf("event count after replay = %d, want 1", count)
	}

	got, err := db.GetEventByClientEventID(ctx, eventID)
	if err != nil {
		t.Fatalf("GetEventByClientEventID: %v", err)
	}
	if got == nil || got.ClientEventID != eventID {
		t.Fatalf("event after replay = %+v", got)
	}
}

func TestOffsetDoesNotAdvanceWhenTransactionAborts(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	firstEventID := mustClientEventID(t, "2026-02-10T18:45:12.123456789Z", 0)
	first := CommitInput{
		Run: testRunInput(),
		Events: []EventInput{
			testEventInput(firstEventID, "2026-02-10T18:45:12.123456789Z", 0),
		},
		File: testFileState(100),
	}
	if err := db.Commit(ctx, first); err != nil {
		t.Fatalf("first Commit: %v", err)
	}

	tx, err := db.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	runID, err := upsertRunTx(ctx, tx, testRunInput())
	if err != nil {
		t.Fatalf("upsertRunTx: %v", err)
	}
	conflict := testEventInput(
		"sha256:0000000000000000000000000000000000000000000000000000000000000001",
		"2026-02-10T18:45:12.123456789Z",
		0,
	)
	_, _, err = insertEventTx(ctx, tx, runID, conflict)
	if err == nil {
		t.Fatal("expected duplicate ordinal failure")
	}

	if err := upsertFileStateTx(ctx, tx, testFileState(999)); err != nil {
		t.Fatalf("upsertFileStateTx: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	state, err := db.GetFileState(ctx, first.File.Path, first.File.FileIdentity)
	if err != nil {
		t.Fatalf("GetFileState: %v", err)
	}
	if state.ByteOffset != 100 {
		t.Fatalf("byte offset after aborted tx = %d, want 100", state.ByteOffset)
	}
}

func TestPartialResumeVersusFullRescan(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	firstID := mustClientEventID(t, "2026-02-10T18:45:12.123456789Z", 0)
	secondID := mustClientEventID(t, "2026-02-10T19:02:33.987654321Z", 0)

	partialFirst := CommitInput{
		Run: testRunInput(),
		Events: []EventInput{
			testEventInput(firstID, "2026-02-10T18:45:12.123456789Z", 0),
		},
		File: testFileState(50),
	}
	if err := db.Commit(ctx, partialFirst); err != nil {
		t.Fatalf("partial first Commit: %v", err)
	}

	partialSecond := CommitInput{
		Run: testRunInput(),
		Events: []EventInput{
			testEventInput(secondID, "2026-02-10T19:02:33.987654321Z", 0),
		},
		File: testFileState(100),
	}
	if err := db.Commit(ctx, partialSecond); err != nil {
		t.Fatalf("partial second Commit: %v", err)
	}

	run, err := db.GetRunByClientRunID(ctx, testRunInput().ClientRunID)
	if err != nil {
		t.Fatalf("GetRunByClientRunID: %v", err)
	}
	partialEvents, err := db.ListEventsByRunID(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListEventsByRunID partial: %v", err)
	}
	if len(partialEvents) != 2 {
		t.Fatalf("partial event count = %d, want 2", len(partialEvents))
	}

	db2 := openTestDB(t)
	defer db2.Close()

	fullRescan := CommitInput{
		Run: testRunInput(),
		Events: []EventInput{
			testEventInput(firstID, "2026-02-10T18:45:12.123456789Z", 0),
			testEventInput(secondID, "2026-02-10T19:02:33.987654321Z", 0),
		},
		File: testFileState(100),
	}
	if err := db2.Commit(ctx, fullRescan); err != nil {
		t.Fatalf("full rescan Commit: %v", err)
	}

	run2, err := db2.GetRunByClientRunID(ctx, testRunInput().ClientRunID)
	if err != nil {
		t.Fatalf("GetRunByClientRunID full: %v", err)
	}
	fullEvents, err := db2.ListEventsByRunID(ctx, run2.ID)
	if err != nil {
		t.Fatalf("ListEventsByRunID full: %v", err)
	}
	if len(fullEvents) != 2 {
		t.Fatalf("full rescan event count = %d, want 2", len(fullEvents))
	}

	if fullEvents[0].ClientEventID != partialEvents[0].ClientEventID {
		t.Fatalf("first event id mismatch: full=%q partial=%q", fullEvents[0].ClientEventID, partialEvents[0].ClientEventID)
	}
	if fullEvents[1].ClientEventID != partialEvents[1].ClientEventID {
		t.Fatalf("second event id mismatch: full=%q partial=%q", fullEvents[1].ClientEventID, partialEvents[1].ClientEventID)
	}
}

func TestSettingsPersistInstallationIDAndTimezone(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	if err := db.SetSetting(ctx, SettingInstallationID, "install-abc"); err != nil {
		t.Fatalf("SetSetting installation_id: %v", err)
	}
	if err := db.SetSetting(ctx, SettingWoWLogTimezone, "Europe/Copenhagen"); err != nil {
		t.Fatalf("SetSetting wow_log_timezone: %v", err)
	}

	installationID, err := db.GetSetting(ctx, SettingInstallationID)
	if err != nil {
		t.Fatalf("GetSetting installation_id: %v", err)
	}
	if installationID != "install-abc" {
		t.Fatalf("installation_id = %q", installationID)
	}

	timezone, err := db.GetSetting(ctx, SettingWoWLogTimezone)
	if err != nil {
		t.Fatalf("GetSetting wow_log_timezone: %v", err)
	}
	if timezone != "Europe/Copenhagen" {
		t.Fatalf("wow_log_timezone = %q", timezone)
	}
}

func TestOpenConfiguresWALAndForeignKeys(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	var journalMode string
	if err := db.SQL().QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	var foreignKeys int
	if err := db.SQL().QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}
}

func TestDuplicateClientEventIDRejectedInTransaction(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	eventID := mustClientEventID(t, "2026-02-10T18:45:12.123456789Z", 0)

	tx, err := db.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	runID, err := upsertRunTx(ctx, tx, testRunInput())
	if err != nil {
		t.Fatalf("upsertRunTx: %v", err)
	}

	first := testEventInput(eventID, "2026-02-10T18:45:12.123456789Z", 0)
	_, inserted, err := insertEventTx(ctx, tx, runID, first)
	if err != nil || !inserted {
		t.Fatalf("first insertEventTx: inserted=%v err=%v", inserted, err)
	}

	second := testEventInput(eventID, "2026-02-10T19:02:33.987654321Z", 1)
	_, inserted, err = insertEventTx(ctx, tx, runID, second)
	if err != nil {
		t.Fatalf("second insertEventTx should ignore duplicate client_event_id: %v", err)
	}
	if inserted {
		t.Fatal("duplicate client_event_id should not insert a second row")
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	count, err := db.CountEvents(ctx)
	if err != nil {
		t.Fatalf("CountEvents: %v", err)
	}
	if count != 1 {
		t.Fatalf("event count = %d, want 1", count)
	}
}
