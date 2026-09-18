package session

import (
	"testing"
	"time"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

func testManager(t *testing.T) (*Manager, *time.Time) {
	t.Helper()
	now := time.Date(2026, 1, 15, 20, 0, 0, 0, time.UTC)
	current := now
	m := NewManager(time.Minute)
	m.now = func() time.Time { return current }
	return m, &current
}

func metadataEvent(line string) parser.Event {
	state := &parser.ParserState{}
	_ = parser.ParseLine(1, "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1", state)
	return parser.ParseLine(2, line, state)
}

func TestManagerStartActivatesRun(t *testing.T) {
	m, _ := testManager(t)
	m.Observe(metadataEvent(`1/15/2026 20:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`), time.UTC)

	if m.State() != RunStateActive {
		t.Fatalf("state = %q, want active", m.State())
	}
	current := m.Current()
	if current == nil || current.ClientRunID == "" {
		t.Fatal("expected active run with client_run_id")
	}
	if current.ChallengeModeStartInstant != "2026-01-15T20:00:00.000000000Z" {
		t.Fatalf("start instant = %q", current.ChallengeModeStartInstant)
	}
}

func TestManagerSuccessfulEndCompletesRun(t *testing.T) {
	m, _ := testManager(t)
	state := &parser.ParserState{}
	lines := []string{
		"COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1",
		`1/15/2026 20:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`,
		`1/15/2026 20:30:00.0000 CHALLENGE_MODE_END,501,1,12,1800000,1800,1800`,
	}
	for i, line := range lines {
		event := parser.ParseLine(i+1, line, state)
		m.Observe(event, time.UTC)
	}

	if m.State() != RunStateCompleted {
		t.Fatalf("state = %q, want completed", m.State())
	}
	if completed := m.TakeCompleted(); completed == nil {
		t.Fatal("expected completed snapshot")
	}
}

func TestManagerCloseNeverCompletesActiveRun(t *testing.T) {
	m, _ := testManager(t)
	m.Observe(metadataEvent(`1/15/2026 20:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`), time.UTC)

	abandoned := m.Close()
	if abandoned == nil || abandoned.State != RunStateAbandoned {
		t.Fatalf("Close() = %+v, want abandoned", abandoned)
	}
	if m.State() != RunStateAbandoned {
		t.Fatalf("state after close = %q", m.State())
	}
}

func TestManagerInactivityAbandonsActiveRun(t *testing.T) {
	m, current := testManager(t)
	m.Observe(metadataEvent(`1/15/2026 20:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`), time.UTC)

	*current = current.Add(2 * time.Minute)
	if abandoned := m.CheckInactivity(); abandoned == nil || abandoned.State != RunStateAbandoned {
		t.Fatalf("CheckInactivity() = %+v, want abandoned", abandoned)
	}
}

func TestManagerMissingTimezoneStartsCandidate(t *testing.T) {
	m, _ := testManager(t)
	m.Observe(metadataEvent(`1/15/2026 20:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`), nil)

	if m.State() != RunStateCandidate {
		t.Fatalf("state = %q, want candidate", m.State())
	}
	if m.Current().ClientRunID != "" {
		t.Fatal("candidate run must not have client_run_id")
	}
}

func TestManagerNewStartAbandonsPreviousRun(t *testing.T) {
	m, _ := testManager(t)
	m.Observe(metadataEvent(`1/15/2026 20:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`), time.UTC)
	firstID := m.Current().ClientRunID

	m.Observe(metadataEvent(`1/15/2026 21:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`), time.UTC)
	if m.State() != RunStateActive {
		t.Fatalf("state = %q, want active", m.State())
	}
	if m.Current().ClientRunID == firstID {
		t.Fatal("expected new run after overlapping start")
	}

	abandoned := m.TakeAbandoned()
	if abandoned == nil {
		t.Fatal("superseded run must be reported for persistence")
	}
	if abandoned.ClientRunID != firstID {
		t.Fatalf("abandoned run = %q, want %q", abandoned.ClientRunID, firstID)
	}
	if abandoned.State != RunStateAbandoned {
		t.Fatalf("abandoned state = %q", abandoned.State)
	}
	if again := m.TakeAbandoned(); again != nil {
		t.Fatalf("TakeAbandoned returned %+v after draining", again)
	}
}

func TestManagerCloseAndInactivityDoNotQueueSupersededRuns(t *testing.T) {
	m, current := testManager(t)
	m.Observe(metadataEvent(`1/15/2026 20:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`), time.UTC)
	if closed := m.Close(); closed == nil {
		t.Fatal("expected Close to return the abandoned run")
	}
	if queued := m.TakeAbandoned(); queued != nil {
		t.Fatalf("Close must not also queue the run: %+v", queued)
	}

	m, current = testManager(t)
	m.Observe(metadataEvent(`1/15/2026 20:00:00.0000 CHALLENGE_MODE_START,"Synthetic Dungeon",501,100,12,[1]`), time.UTC)
	*current = current.Add(2 * time.Minute)
	if abandoned := m.CheckInactivity(); abandoned == nil {
		t.Fatal("expected CheckInactivity to return the abandoned run")
	}
	if queued := m.TakeAbandoned(); queued != nil {
		t.Fatalf("CheckInactivity must not also queue the run: %+v", queued)
	}
}
