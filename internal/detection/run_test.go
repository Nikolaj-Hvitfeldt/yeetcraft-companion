package detection

import (
	"os"
	"testing"
	"time"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

func testLogTimezone(t *testing.T) *time.Location {
	t.Helper()
	return time.UTC
}

func scanFixture(t *testing.T, path string, guids []string) *Tracker {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	tracker, err := NewTracker(guids, testLogTimezone(t))
	if err != nil {
		t.Fatal(err)
	}
	state := &parser.ParserState{}
	if _, err := parser.ScanReader(file, parser.DefaultMaxLineSize, state, tracker.Observe); err != nil {
		t.Fatal(err)
	}
	return tracker
}

func TestRunTrackerCapturesStartInstant(t *testing.T) {
	lines := []string{
		"COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1",
		"1/15/2026 20:04:00.0000 CHALLENGE_MODE_START,\"Synthetic Dungeon\",501,100,12,[1]",
	}
	tracker, err := NewTracker([]string{"Player-9999-00000001"}, testLogTimezone(t))
	if err != nil {
		t.Fatal(err)
	}
	state := &parser.ParserState{}
	for i, line := range lines {
		if err := tracker.Observe(parser.ParseLine(i+1, line, state)); err != nil {
			t.Fatal(err)
		}
	}

	runCtx := tracker.run.context()
	if !runCtx.Active {
		t.Fatal("expected active run")
	}
	if runCtx.StartInstant != "2026-01-15T20:04:00.000000000Z" {
		t.Fatalf("start instant = %q", runCtx.StartInstant)
	}
	if runCtx.StartInstantHold != "" {
		t.Fatalf("start instant hold = %q", runCtx.StartInstantHold)
	}
}

func TestDeathBeforeChallengeModeStartHeld(t *testing.T) {
	const path = "../../testdata/logs/synthetic/death-outside-run.txt"
	tracker := scanFixture(t, path, []string{"Player-9999-00000002"})

	deaths := tracker.Deaths()
	if len(deaths) != 3 {
		t.Fatalf("deaths = %d, want 3", len(deaths))
	}

	before := deaths[0]
	if before.HoldReason != DeathHoldRunContextIncomplete {
		t.Fatalf("before-start hold = %q, want %q", before.HoldReason, DeathHoldRunContextIncomplete)
	}
	if before.CanComputeIDs() {
		t.Fatal("death before CHALLENGE_MODE_START must not produce IDs")
	}
	if _, err := before.ClientRunID(); err == nil {
		t.Fatal("ClientRunID should fail for held death")
	}

	during := deaths[1]
	if during.HoldReason != "" {
		t.Fatalf("during-run hold = %q, want none", during.HoldReason)
	}
	if !during.CanComputeIDs() {
		t.Fatalf("death during active run must produce IDs, hold=%q instantHold=%q", during.HoldReason, during.DeathInstantHold)
	}
}

func TestDeathAfterChallengeModeEndHeld(t *testing.T) {
	const path = "../../testdata/logs/synthetic/death-outside-run.txt"
	tracker := scanFixture(t, path, []string{"Player-9999-00000002"})

	after := tracker.Deaths()[2]
	if after.HoldReason != DeathHoldRunContextIncomplete {
		t.Fatalf("after-end hold = %q, want %q", after.HoldReason, DeathHoldRunContextIncomplete)
	}
	if after.CanComputeIDs() {
		t.Fatal("death after CHALLENGE_MODE_END must not produce IDs")
	}
}

func TestSameInstantDeathOrdinals(t *testing.T) {
	const path = "../../testdata/logs/synthetic/same-instant-deaths.txt"
	tracker := scanFixture(t, path, []string{"Player-9999-00000001"})

	deaths := tracker.Deaths()
	if len(deaths) != 2 {
		t.Fatalf("deaths = %d, want 2", len(deaths))
	}
	if deaths[0].DeathInstant != "2026-01-15T20:04:10.000000000Z" {
		t.Fatalf("death instant = %q", deaths[0].DeathInstant)
	}
	if !deaths[0].CanComputeIDs() || !deaths[1].CanComputeIDs() {
		t.Fatal("same-instant deaths during active run must be hashable")
	}
	if deaths[0].Ordinal != 0 || deaths[1].Ordinal != 1 {
		t.Fatalf("ordinals = [%d, %d], want [0, 1]", deaths[0].Ordinal, deaths[1].Ordinal)
	}
	if deaths[0].OrdinalAssigned != true || deaths[1].OrdinalAssigned != true {
		t.Fatal("expected ordinals to be assigned")
	}

	runID0, err := deaths[0].ClientRunID()
	if err != nil {
		t.Fatalf("ClientRunID: %v", err)
	}
	eventID0, err := deaths[0].ClientEventID()
	if err != nil {
		t.Fatalf("ClientEventID death 0: %v", err)
	}
	eventID1, err := deaths[1].ClientEventID()
	if err != nil {
		t.Fatalf("ClientEventID death 1: %v", err)
	}
	if eventID0 == eventID1 {
		t.Fatal("same-instant deaths must produce distinct clientEventId values")
	}
	if runID0 == "" {
		t.Fatal("expected clientRunId")
	}
}

func TestRepeatedDeathAfterResurrectionResetsOrdinal(t *testing.T) {
	lines := []string{
		"COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1",
		"1/15/2026 20:08:00.0000 CHALLENGE_MODE_START,\"Synthetic Dungeon\",501,100,12,[1]",
		"1/15/2026 20:08:10.0000 UNIT_DIED,0000000000000000,nil,0x80000000,0x80000000,Player-9999-00000001,\"TrackedAlpha-SyntheticRealm\",0x512,0x0,0,0",
		"1/15/2026 20:09:10.0000 UNIT_DIED,0000000000000000,nil,0x80000000,0x80000000,Player-9999-00000001,\"TrackedAlpha-SyntheticRealm\",0x512,0x0,0,0",
	}
	tracker, err := NewTracker([]string{"Player-9999-00000001"}, testLogTimezone(t))
	if err != nil {
		t.Fatal(err)
	}
	state := &parser.ParserState{}
	for i, line := range lines {
		if err := tracker.Observe(parser.ParseLine(i+1, line, state)); err != nil {
			t.Fatal(err)
		}
	}

	deaths := tracker.Deaths()
	if len(deaths) != 2 {
		t.Fatalf("deaths = %d, want 2", len(deaths))
	}
	if deaths[0].Ordinal != 0 || deaths[1].Ordinal != 0 {
		t.Fatalf("ordinals = [%d, %d], want [0, 0] at different instants", deaths[0].Ordinal, deaths[1].Ordinal)
	}
	if deaths[0].DeathInstant == deaths[1].DeathInstant {
		t.Fatal("expected different canonical death instants")
	}
}

func TestPlayerSourceCauseRedacted(t *testing.T) {
	const path = "../../testdata/logs/synthetic/player-source-cause.txt"
	tracker := scanFixture(t, path, []string{"Player-9999-00000001"})

	deaths := tracker.Deaths()
	if len(deaths) != 1 {
		t.Fatalf("deaths = %d, want 1", len(deaths))
	}
	if len(deaths[0].Causes) < 2 {
		t.Fatalf("causes = %d, want at least 2", len(deaths[0].Causes))
	}

	var playerFound bool
	var creatureFound bool
	for _, cause := range deaths[0].Causes {
		if cause.Cause.PlayerOrigin {
			playerFound = true
			if cause.Cause.HasCreatureID {
				t.Fatal("player-origin cause must not carry creatureId")
			}
		}
		if cause.Cause.HasCreatureID {
			creatureFound = true
			if cause.Cause.PlayerOrigin {
				t.Fatal("creature cause must not be marked player-origin")
			}
			if cause.Cause.CreatureID != 900013 {
				t.Fatalf("creature id = %d, want 900013", cause.Cause.CreatureID)
			}
		}
	}
	if !playerFound {
		t.Fatal("expected player-origin cause to be redacted with boolean marker")
	}
	if !creatureFound {
		t.Fatal("expected creature cause with creatureId")
	}
}
