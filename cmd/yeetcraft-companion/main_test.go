package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/logwatcher"
	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/session"
	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/storage"
)

const (
	testTrackedGUID = "Player-9999-00000001"
	testTimezone    = "UTC"
)

func testLookup(extra map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		switch key {
		case "YEETCRAFT_TRACKED_GUIDS":
			if value, ok := extra["YEETCRAFT_TRACKED_GUIDS"]; ok {
				return value, ok
			}
			return testTrackedGUID, true
		case "WOW_LOG_TIMEZONE":
			if value, ok := extra["WOW_LOG_TIMEZONE"]; ok {
				return value, ok
			}
			return testTimezone, true
		default:
			value, ok := extra[key]
			return value, ok
		}
	}
}

func writeTestLog(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func appendTestLog(t *testing.T, path, content string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		t.Fatal(err)
	}
}

func runOnce(t *testing.T, logPath, dbPath string, lookup func(string) (string, bool)) int {
	t.Helper()
	return runCapture(context.Background(), captureOptions{
		LogFile:           logPath,
		DBPath:            dbPath,
		Once:              true,
		InactivityTimeout: time.Minute,
		Lookup:            lookup,
	}, os.Stderr)
}

func expectedSpellDamageIDs(t *testing.T) (string, string) {
	t.Helper()
	runID, err := session.ClientRunID(session.RunIDParams{
		ChallengeModeStartInstant: "2026-01-15T20:00:00.000000000Z",
		ChallengeMapID:            501,
		KeystoneLevel:             12,
	})
	if err != nil {
		t.Fatalf("ClientRunID: %v", err)
	}
	eventID, err := session.ClientEventID(session.EventIDParams{
		ClientRunID:   runID,
		CharacterGUID: testTrackedGUID,
		DeathInstant:  "2026-01-15T20:00:01.000100000Z",
		Ordinal:       0,
	})
	if err != nil {
		t.Fatalf("ClientEventID: %v", err)
	}
	return runID, eventID
}

func TestMissingTrackedConfigExitsNonZeroAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	logPath := writeTestLog(t, dir, "WoWCombatLog.txt", "line\n")
	dbPath := filepath.Join(dir, "companion.db")

	code := runOnce(t, logPath, dbPath, testLookup(map[string]string{
		"YEETCRAFT_TRACKED_GUIDS": "",
	}))
	if code != exitConfig {
		t.Fatalf("exit code = %d, want %d", code, exitConfig)
	}
	if _, err := os.Stat(dbPath); err == nil {
		t.Fatal("database file should not be created when config fails closed")
	}
}

func TestSyntheticLogProducesExpectedEventCountAndIDs(t *testing.T) {
	dir := t.TempDir()
	logPath := writeTestLog(t, dir, "WoWCombatLog.txt", readFixture(t, "../../testdata/logs/synthetic/spell-damage-death.txt"))
	dbPath := filepath.Join(dir, "companion.db")

	if code := runOnce(t, logPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	count, err := db.CountEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("event count = %d, want 1", count)
	}

	wantRunID, wantEventID := expectedSpellDamageIDs(t)
	run, err := db.GetRunByClientRunID(ctx, wantRunID)
	if err != nil {
		t.Fatal(err)
	}
	if run == nil {
		t.Fatalf("run %q not persisted", wantRunID)
	}
	if run.Status != storage.RunStatusActive {
		t.Fatalf("run status = %q, want active after --once poll", run.Status)
	}

	event, err := db.GetEventByClientEventID(ctx, wantEventID)
	if err != nil {
		t.Fatal(err)
	}
	if event == nil {
		t.Fatalf("event %q not persisted", wantEventID)
	}
	if event.CharacterGUID != testTrackedGUID || event.Ordinal != 0 {
		t.Fatalf("event = %+v", event)
	}
}

func TestRestartMidLogSameIDsNoDuplicates(t *testing.T) {
	dir := t.TempDir()
	full := readFixture(t, "../../testdata/logs/synthetic/spell-damage-death.txt")
	split := strings.Index(full, "UNIT_DIED")
	if split < 0 {
		t.Fatal("could not find split point in fixture")
	}
	partial := full[:split]
	remainder := full[split:]

	writeTestLog(t, dir, "WoWCombatLog.txt", partial)
	logPath := filepath.Join(dir, "WoWCombatLog.txt")
	dbPath := filepath.Join(dir, "companion.db")

	if code := runOnce(t, logPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("first run exit = %d", code)
	}
	appendTestLog(t, logPath, remainder)
	if code := runOnce(t, logPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("second run exit = %d", code)
	}
	if code := runOnce(t, logPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("third run exit = %d", code)
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	count, err := db.CountEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("event count = %d, want 1", count)
	}

	_, wantEventID := expectedSpellDamageIDs(t)
	event, err := db.GetEventByClientEventID(ctx, wantEventID)
	if err != nil {
		t.Fatal(err)
	}
	if event == nil {
		t.Fatal("expected persisted event after restart")
	}
}

func TestTruncationPersistOnce(t *testing.T) {
	dir := t.TempDir()
	logPath := writeTestLog(t, dir, "WoWCombatLog.txt", readFixture(t, "../../testdata/logs/synthetic/spell-damage-death.txt"))
	dbPath := filepath.Join(dir, "companion.db")

	if code := runOnce(t, logPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("first run exit = %d", code)
	}

	writeTestLog(t, dir, "WoWCombatLog.txt", "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1\n")
	if code := runOnce(t, logPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("truncated run exit = %d", code)
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	count, err := db.CountEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("event count after truncation = %d, want 1", count)
	}

	identity, _, err := logwatcher.ResolveIdentity(logPath)
	if err != nil {
		t.Fatal(err)
	}
	state, err := db.GetFileState(ctx, logPath, identity.StorageKey())
	if err != nil {
		t.Fatal(err)
	}
	if state == nil || state.Generation < 1 {
		t.Fatalf("expected truncation to advance generation, state=%+v", state)
	}
}

func TestRotationPersistOnce(t *testing.T) {
	dir := t.TempDir()
	oldPath := writeTestLog(t, dir, "WoWCombatLog-old.txt", "old-line\n")
	newPath := writeTestLog(t, dir, "WoWCombatLog-new.txt", readFixture(t, "../../testdata/logs/synthetic/spell-damage-death.txt"))
	dbPath := filepath.Join(dir, "companion.db")

	if code := runOnce(t, oldPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("old log run exit = %d", code)
	}
	if code := runOnce(t, newPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("new log run exit = %d", code)
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	count, err := db.CountEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("event count = %d, want 1", count)
	}
}

// A key left before it completes is followed by another CHALLENGE_MODE_START.
// The superseded run must not stay active in the database.
func TestSupersededRunIsPersistedAsAbandoned(t *testing.T) {
	dir := t.TempDir()
	log := strings.Join([]string{
		"COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1",
		`1/15/2026 20:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`,
		`1/15/2026 21:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`,
		"",
	}, "\n")
	logPath := writeTestLog(t, dir, "WoWCombatLog.txt", log)
	dbPath := filepath.Join(dir, "companion.db")

	if code := runOnce(t, logPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	firstRunID, err := session.ClientRunID(session.RunIDParams{
		ChallengeModeStartInstant: "2026-01-15T20:00:00.000000000Z",
		ChallengeMapID:            501,
		KeystoneLevel:             12,
	})
	if err != nil {
		t.Fatal(err)
	}

	first, err := db.GetRunByClientRunID(ctx, firstRunID)
	if err != nil {
		t.Fatal(err)
	}
	if first == nil {
		t.Fatalf("superseded run %q was not persisted", firstRunID)
	}
	if first.Status != storage.RunStatusAbandoned {
		t.Fatalf("superseded run status = %q, want %q", first.Status, storage.RunStatusAbandoned)
	}
	if first.EndedAt == nil {
		t.Fatal("superseded run must record ended_at")
	}

	active, err := db.GetActiveRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ClientRunID == firstRunID {
		t.Fatalf("active run = %+v, want the second run", active)
	}
}

func TestDeathThenSuccessfulEndPersistsUnderThatRun(t *testing.T) {
	dir := t.TempDir()
	log := strings.TrimRight(readFixture(t, "../../testdata/logs/synthetic/spell-damage-death.txt"), "\n") + "\n" +
		`1/15/2026 20:30:00.0000 CHALLENGE_MODE_END,501,1,12,1800000,-12.5,1800.000000` + "\n"
	logPath := writeTestLog(t, dir, "WoWCombatLog.txt", log)
	dbPath := filepath.Join(dir, "companion.db")

	if code := runOnce(t, logPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	wantRunID, wantEventID := expectedSpellDamageIDs(t)
	run, err := db.GetRunByClientRunID(ctx, wantRunID)
	if err != nil {
		t.Fatal(err)
	}
	if run == nil {
		t.Fatal("run not persisted")
	}
	if run.Status != storage.RunStatusCompleted {
		t.Fatalf("run status = %q, want completed", run.Status)
	}

	event, err := db.GetEventByClientEventID(ctx, wantEventID)
	if err != nil {
		t.Fatal(err)
	}
	if event == nil || event.RunID != run.ID {
		t.Fatalf("event = %+v, want run id %d", event, run.ID)
	}
}

func TestDeathThenNewStartStaysOnFirstRun(t *testing.T) {
	dir := t.TempDir()
	log := strings.TrimRight(readFixture(t, "../../testdata/logs/synthetic/spell-damage-death.txt"), "\n") + "\n" +
		`1/15/2026 21:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]` + "\n"
	logPath := writeTestLog(t, dir, "WoWCombatLog.txt", log)
	dbPath := filepath.Join(dir, "companion.db")

	if code := runOnce(t, logPath, dbPath, testLookup(nil)); code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	firstRunID, wantEventID := expectedSpellDamageIDs(t)
	first, err := db.GetRunByClientRunID(ctx, firstRunID)
	if err != nil {
		t.Fatal(err)
	}
	if first == nil {
		t.Fatal("first run not persisted")
	}
	if first.Status != storage.RunStatusAbandoned {
		t.Fatalf("first run status = %q, want abandoned", first.Status)
	}

	event, err := db.GetEventByClientEventID(ctx, wantEventID)
	if err != nil {
		t.Fatal(err)
	}
	if event == nil || event.RunID != first.ID {
		t.Fatalf("death attached to run id %v, want first run %d", event, first.ID)
	}

	secondRunID, err := session.ClientRunID(session.RunIDParams{
		ChallengeModeStartInstant: "2026-01-15T21:00:00.000000000Z",
		ChallengeMapID:            501,
		KeystoneLevel:             12,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.GetRunByClientRunID(ctx, secondRunID)
	if err != nil {
		t.Fatal(err)
	}
	if second == nil || second.Status != storage.RunStatusActive {
		t.Fatalf("second run = %+v, want active", second)
	}
}

func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
