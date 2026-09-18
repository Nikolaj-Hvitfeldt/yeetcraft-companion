package detection

import (
	"errors"
	"os"
	"testing"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

func TestNewTrackerRequiresGUIDs(t *testing.T) {
	tests := []struct {
		name    string
		guids   []string
		wantErr error
	}{
		{name: "nil", guids: nil, wantErr: ErrNoTrackedGUIDs},
		{name: "empty", guids: []string{}, wantErr: ErrNoTrackedGUIDs},
		{name: "blank entry", guids: []string{""}, wantErr: ErrNoTrackedGUIDs},
		{name: "invalid shape", guids: []string{"not-a-guid"}, wantErr: ErrInvalidTrackedGUID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker, err := NewTracker(tt.guids)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewTracker() error = %v, want %v", err, tt.wantErr)
				}
				if tracker != nil {
					t.Fatal("expected nil tracker on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewTracker() unexpected error: %v", err)
			}
		})
	}
}

func TestNewDiagnosticTrackerTracksAllPlayers(t *testing.T) {
	tracker := NewDiagnosticTracker()
	if tracker == nil {
		t.Fatal("NewDiagnosticTracker returned nil")
	}
	if !tracker.tracks("Player-9999-00000099") {
		t.Fatal("diagnostic tracker should track any player guid")
	}
}

func TestTrackerBossContextDeath(t *testing.T) {
	const path = "../../testdata/logs/synthetic/boss-context-death.txt"
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	tracker, err := NewTracker([]string{"Player-9999-00000004"})
	if err != nil {
		t.Fatal(err)
	}
	state := &parser.ParserState{}
	_, err = parser.ScanReader(file, parser.DefaultMaxLineSize, state, tracker.Observe)
	if err != nil {
		t.Fatal(err)
	}

	deaths := tracker.Deaths()
	if len(deaths) != 1 {
		t.Fatalf("deaths = %d, want 1", len(deaths))
	}
	death := deaths[0]
	if death.VictimGUID != "Player-9999-00000004" {
		t.Fatalf("victim = %q", death.VictimGUID)
	}
	if !death.Encounter.Active || death.Encounter.EncounterID != 990001 {
		t.Fatalf("encounter = %+v", death.Encounter)
	}
	if death.Encounter.EncounterName != "Synthetic Colossus" {
		t.Fatalf("encounter name = %q", death.Encounter.EncounterName)
	}
	if len(death.Causes) == 0 {
		t.Fatal("expected at least one cause")
	}
	cause := death.Causes[0]
	if cause.Hit.SpellName != "Synthetic Collapse" || cause.Hit.SpellID != 900003 {
		t.Fatalf("primary cause = %+v", cause.Hit)
	}
	if cause.Confidence != ConfidenceHigh {
		t.Fatalf("confidence = %q, want high (overkill present)", cause.Confidence)
	}
}

func TestTrackerSpellDamageDeath(t *testing.T) {
	const path = "../../testdata/logs/synthetic/spell-damage-death.txt"
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	tracker, err := NewTracker([]string{"Player-9999-00000001"})
	if err != nil {
		t.Fatal(err)
	}
	state := &parser.ParserState{}
	if _, err := parser.ScanReader(file, parser.DefaultMaxLineSize, state, tracker.Observe); err != nil {
		t.Fatal(err)
	}
	deaths := tracker.Deaths()
	if len(deaths) != 1 {
		t.Fatalf("deaths = %d, want 1", len(deaths))
	}
	if deaths[0].Encounter.Active {
		t.Fatalf("expected trash death without active encounter")
	}
}

func TestTrackerIgnoresUntrackedPlayer(t *testing.T) {
	const path = "../../testdata/logs/synthetic/untracked-party-death.txt"
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	tracker, err := NewTracker([]string{"Player-9999-00000001"})
	if err != nil {
		t.Fatal(err)
	}
	state := &parser.ParserState{}
	if _, err := parser.ScanReader(file, parser.DefaultMaxLineSize, state, tracker.Observe); err != nil {
		t.Fatal(err)
	}
	if len(tracker.Deaths()) != 0 {
		t.Fatalf("expected untracked death to be ignored, got %d", len(tracker.Deaths()))
	}
}

func TestTrackerChallengeRunContext(t *testing.T) {
	lines := []string{
		"COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1",
		"CHALLENGE_MODE_START,\"Synthetic Dungeon\",501,100,12,[1]",
		"ENCOUNTER_START,9001,\"Synthetic Boss\",8,5",
		"SPELL_DAMAGE,Creature-Synthetic,\"Synthetic Source\",0xa48,0x0,Player-9999-00000001,\"TrackedAlpha-SyntheticRealm\",0x512,0x0,900001,\"Synthetic Bolt\",0x40,Player-9999-00000001,0000000000000000,0,100000,0,50000,1000,0,0,0,0,100,100,0,10.00,20.00,9999,0.0000,500,120000,120000,20000,0x40,0,0,0,nil,nil,nil,ST",
		"UNIT_DIED,0000000000000000,nil,0x80000000,0x80000000,Player-9999-00000001,\"TrackedAlpha-SyntheticRealm\",0x512,0x0,0",
	}
	tracker, err := NewTracker([]string{"Player-9999-00000001"})
	if err != nil {
		t.Fatal(err)
	}
	state := &parser.ParserState{}
	for i, line := range lines {
		if err := tracker.Observe(parser.ParseLine(i+1, line, state)); err != nil {
			t.Fatal(err)
		}
	}
	death := tracker.Deaths()[0]
	if !death.Run.Active || death.Run.MapID != 501 || death.Run.KeystoneLevel != 12 {
		t.Fatalf("run = %+v", death.Run)
	}
	if death.Run.DungeonName != "Synthetic Dungeon" {
		t.Fatalf("dungeon = %q", death.Run.DungeonName)
	}
	if !death.Encounter.Active || death.Encounter.EncounterID != 9001 {
		t.Fatalf("encounter = %+v", death.Encounter)
	}
}

func TestTrackerClearsEncounterAfterEnd(t *testing.T) {
	lines := []string{
		"COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1",
		"ENCOUNTER_START,9001,\"Synthetic Boss\",8,5",
		"ENCOUNTER_END,9001,\"Synthetic Boss\",8,5,1",
		"UNIT_DIED,0000000000000000,nil,0x80000000,0x80000000,Player-9999-00000001,\"TrackedAlpha-SyntheticRealm\",0x512,0x0,0",
	}
	tracker, err := NewTracker([]string{"Player-9999-00000001"})
	if err != nil {
		t.Fatal(err)
	}
	state := &parser.ParserState{}
	for i, line := range lines {
		if err := tracker.Observe(parser.ParseLine(i+1, line, state)); err != nil {
			t.Fatal(err)
		}
	}

	death := tracker.Deaths()[0]
	if death.Encounter.Active || death.Encounter.EncounterID != 0 || death.Encounter.EncounterName != "" {
		t.Fatalf("encounter = %+v, want no active encounter", death.Encounter)
	}
}
