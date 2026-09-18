package detection

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

// ErrNoTrackedGUIDs is returned when NewTracker is called with an empty or nil GUID list.
var ErrNoTrackedGUIDs = errors.New("tracked guids required")

// ErrInvalidTrackedGUID is returned when a tracked GUID does not match the Player- prefix shape.
var ErrInvalidTrackedGUID = errors.New("invalid tracked player guid")

var playerGUIDPattern = regexp.MustCompile(`^Player-[0-9A-Za-z]+-[0-9A-Za-z]+$`)

// Tracker consumes parsed combat-log events and accumulates death candidates.
type Tracker struct {
	tracked    map[string]struct{}
	allPlayers bool

	run       runTracker
	encounter encounterTracker
	buffers   map[string]*damageBuffer
	deaths    []DeathCandidate
}

// NewTracker returns a tracker that records deaths only for the listed player GUIDs.
// An empty or nil list is an error; production capture must never default to track-all.
func NewTracker(trackedGUIDs []string) (*Tracker, error) {
	if len(trackedGUIDs) == 0 {
		return nil, ErrNoTrackedGUIDs
	}
	t := &Tracker{
		buffers: make(map[string]*damageBuffer),
		tracked: make(map[string]struct{}, len(trackedGUIDs)),
	}
	for _, guid := range trackedGUIDs {
		if guid == "" {
			return nil, ErrNoTrackedGUIDs
		}
		if !playerGUIDPattern.MatchString(guid) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTrackedGUID, guid)
		}
		t.tracked[guid] = struct{}{}
	}
	return t, nil
}

// NewDiagnosticTracker returns a tracker that records every player death.
// It is the only supported path to track-all behavior and is intended for logprobe diagnostics.
func NewDiagnosticTracker() *Tracker {
	return &Tracker{
		allPlayers: true,
		buffers:    make(map[string]*damageBuffer),
	}
}

// Observe ingests one parsed event.
func (t *Tracker) Observe(event parser.Event) error {
	if event.Kind == parser.KindMetadata {
		t.run.observe(event)
		t.encounter.observe(event)
		return nil
	}
	if event.Kind != parser.KindCommonHeader || event.Common == nil {
		return nil
	}
	if hit, ok := extractDamageHit(event); ok {
		dest := event.Common.DestGUID
		if !isPlayerGUID(dest) {
			return nil
		}
		buf := t.bufferFor(dest)
		buf.add(hit)
	}
	if event.EventType == "UNIT_DIED" {
		t.observeDeath(event)
	}
	return nil
}

// Deaths returns accumulated candidates in encounter order.
func (t *Tracker) Deaths() []DeathCandidate {
	if len(t.deaths) == 0 {
		return nil
	}
	out := make([]DeathCandidate, len(t.deaths))
	copy(out, t.deaths)
	return out
}

func (t *Tracker) bufferFor(guid string) *damageBuffer {
	buf, ok := t.buffers[guid]
	if !ok {
		buf = newDamageBuffer(defaultDamageCapacity)
		t.buffers[guid] = buf
	}
	return buf
}

func (t *Tracker) observeDeath(event parser.Event) {
	victim := event.Common.DestGUID
	if !isPlayerGUID(victim) || !t.tracks(victim) {
		return
	}
	var hits []DamageHit
	if buf := t.buffers[victim]; buf != nil {
		hits = buf.snapshotBefore(event.LineNumber)
	}
	t.deaths = append(t.deaths, DeathCandidate{
		LineNumber: event.LineNumber,
		Timestamp:  event.Envelope.Raw,
		VictimGUID: victim,
		Run:        t.run.context(),
		Encounter:  t.encounter.context(),
		Causes:     rankCauses(hits),
	})
}

func (t *Tracker) tracks(guid string) bool {
	if t.allPlayers {
		return true
	}
	_, ok := t.tracked[guid]
	return ok
}
