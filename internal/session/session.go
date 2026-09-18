package session

import (
	"time"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

// DefaultInactivityTimeout closes an active run as abandoned when no log activity
// arrives for this duration. Real-log abandonment signals remain evidence backlog.
const DefaultInactivityTimeout = 30 * time.Minute

// RunState is the local Mythic+ run lifecycle state.
type RunState string

const (
	RunStateIdle       RunState = "idle"
	RunStateCandidate  RunState = "candidate"
	RunStateActive     RunState = "active"
	RunStateCompleting RunState = "completing"
	RunStateCompleted  RunState = "completed"
	RunStateAbandoned  RunState = "abandoned"
)

// RunSnapshot is the persisted run boundary derived from the state machine.
type RunSnapshot struct {
	State                     RunState
	ClientRunID               string
	ChallengeModeStartInstant string
	ChallengeMapID            int64
	KeystoneLevel             int64
	DungeonName               string
	StartedAt                 time.Time
	LastActivity              time.Time
	EndedAt                   *time.Time
}

// Manager tracks Mythic+ run boundaries for headless capture.
type Manager struct {
	inactivityTimeout time.Duration
	now               func() time.Time
	current           *RunSnapshot
}

// NewManager returns a run state machine with the given inactivity timeout.
func NewManager(inactivityTimeout time.Duration) *Manager {
	if inactivityTimeout <= 0 {
		inactivityTimeout = DefaultInactivityTimeout
	}
	return &Manager{
		inactivityTimeout: inactivityTimeout,
		now:               time.Now,
	}
}

// State returns the current run state.
func (m *Manager) State() RunState {
	if m.current == nil {
		return RunStateIdle
	}
	return m.current.State
}

// Current returns a copy of the active run snapshot, or nil when idle or terminal.
func (m *Manager) Current() *RunSnapshot {
	if m.current == nil || isTerminal(m.current.State) {
		return nil
	}
	snapshot := *m.current
	return &snapshot
}

// RestoreActiveRun rehydrates an active run after process restart.
func (m *Manager) RestoreActiveRun(clientRunID, startInstant string, mapID, keystone int64, startedAt time.Time) {
	m.current = &RunSnapshot{
		State:                     RunStateActive,
		ClientRunID:               clientRunID,
		ChallengeModeStartInstant: startInstant,
		ChallengeMapID:            mapID,
		KeystoneLevel:             keystone,
		StartedAt:                 startedAt,
		LastActivity:              m.now(),
	}
}

// Observe ingests one parsed combat-log event and updates run state.
func (m *Manager) Observe(event parser.Event, loc *time.Location) {
	if event.Kind != parser.KindMetadata {
		if m.current != nil && (m.current.State == RunStateActive || m.current.State == RunStateCandidate || m.current.State == RunStateCompleting) {
			m.current.LastActivity = m.now()
		}
		return
	}

	switch event.EventType {
	case "CHALLENGE_MODE_START":
		m.observeStart(event, loc)
	case "CHALLENGE_MODE_END":
		m.observeEnd(event)
	}
}

// CheckInactivity closes an active run as abandoned when it has been idle too long.
func (m *Manager) CheckInactivity() *RunSnapshot {
	if m.current == nil {
		return nil
	}
	switch m.current.State {
	case RunStateActive, RunStateCandidate, RunStateCompleting:
	default:
		return nil
	}
	if m.now().Sub(m.current.LastActivity) < m.inactivityTimeout {
		return nil
	}
	return m.closeAsAbandoned()
}

// Close closes any non-terminal run as abandoned. Process exit must never mark a run completed.
func (m *Manager) Close() *RunSnapshot {
	if m.current == nil {
		return nil
	}
	switch m.current.State {
	case RunStateActive, RunStateCandidate, RunStateCompleting:
		return m.closeAsAbandoned()
	default:
		return nil
	}
}

// TakeCompleted returns and clears a run that reached completed state.
func (m *Manager) TakeCompleted() *RunSnapshot {
	if m.current == nil || m.current.State != RunStateCompleted {
		return nil
	}
	snapshot := *m.current
	m.current = nil
	return &snapshot
}

// TakeAbandoned returns and clears a run that reached abandoned state.
func (m *Manager) TakeAbandoned() *RunSnapshot {
	if m.current == nil || m.current.State != RunStateAbandoned {
		return nil
	}
	snapshot := *m.current
	m.current = nil
	return &snapshot
}

func (m *Manager) observeStart(event parser.Event, loc *time.Location) {
	fields := event.Fields
	if len(fields) < 5 {
		return
	}
	mapID, ok := parseInt64(fields[2])
	if !ok {
		return
	}
	keyLevel, ok := parseInt64(fields[4])
	if !ok {
		return
	}

	res := parser.ResolveCanonicalInstant(event.Envelope.Raw, loc)
	now := m.now()

	if m.current != nil && !isTerminal(m.current.State) {
		m.closeAsAbandoned()
	}

	snapshot := &RunSnapshot{
		ChallengeMapID: mapID,
		KeystoneLevel:  keyLevel,
		DungeonName:    fields[1],
		StartedAt:      now,
		LastActivity:   now,
	}
	if res.HoldReason != "" || res.Canonical == "" {
		snapshot.State = RunStateCandidate
		m.current = snapshot
		return
	}

	clientRunID, err := ClientRunID(RunIDParams{
		ChallengeModeStartInstant: res.Canonical,
		ChallengeMapID:            int(mapID),
		KeystoneLevel:             int(keyLevel),
	})
	if err != nil {
		snapshot.State = RunStateCandidate
		m.current = snapshot
		return
	}

	snapshot.State = RunStateActive
	snapshot.ClientRunID = clientRunID
	snapshot.ChallengeModeStartInstant = res.Canonical
	m.current = snapshot
}

func (m *Manager) observeEnd(event parser.Event) {
	if m.current == nil || m.current.State == RunStateIdle || isTerminal(m.current.State) {
		return
	}
	if event.Typed.Status != parser.TypedStatusParsed {
		return
	}
	payload, ok := event.Typed.Payload.(parser.ChallengeModeEndPayload)
	if !ok {
		return
	}
	if !payload.Success && payload.KeystoneLevel == 0 {
		return
	}

	m.current.LastActivity = m.now()
	if !payload.Success {
		return
	}

	m.current.State = RunStateCompleting
	endedAt := m.now()
	m.current.EndedAt = &endedAt
	m.current.State = RunStateCompleted
}

func (m *Manager) closeAsAbandoned() *RunSnapshot {
	if m.current == nil {
		return nil
	}
	endedAt := m.now()
	m.current.EndedAt = &endedAt
	m.current.State = RunStateAbandoned
	snapshot := *m.current
	return &snapshot
}

func isTerminal(state RunState) bool {
	return state == RunStateCompleted || state == RunStateAbandoned
}

func parseInt64(raw string) (int64, bool) {
	if raw == "" {
		return 0, false
	}
	var negative bool
	if raw[0] == '-' {
		negative = true
		raw = raw[1:]
	}
	if raw == "" {
		return 0, false
	}
	var value int64
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if ch < '0' || ch > '9' {
			return 0, false
		}
		value = value*10 + int64(ch-'0')
	}
	if negative {
		value = -value
	}
	return value, true
}
