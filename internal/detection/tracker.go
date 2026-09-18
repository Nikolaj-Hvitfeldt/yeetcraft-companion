package detection

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

// ErrNoTrackedGUIDs is returned when NewTracker is called with an empty or nil GUID list.
var ErrNoTrackedGUIDs = errors.New("tracked guids required")

// ErrInvalidTrackedGUID is returned when a tracked GUID does not match the Player- prefix shape.
var ErrInvalidTrackedGUID = errors.New("invalid tracked player guid")

var playerGUIDPattern = regexp.MustCompile(`^Player-[0-9A-Za-z]+-[0-9A-Za-z]+$`)

type ordinalKey struct {
	victim       string
	deathInstant string
	runStart     string
}

// Tracker consumes parsed combat-log events and accumulates death candidates.
type Tracker struct {
	tracked     map[string]struct{}
	allPlayers  bool
	logTimezone *time.Location

	run       runTracker
	encounter encounterTracker
	buffers   map[string]*damageBuffer
	ordinals  map[ordinalKey]int
	deaths    []DeathCandidate
}

// NewTracker returns a tracker that records deaths only for the listed player GUIDs.
// An empty or nil list is an error; production capture must never default to track-all.
// logTimezone resolves timezone-less envelope stamps; nil yields missing_log_timezone holds.
func NewTracker(trackedGUIDs []string, logTimezone *time.Location) (*Tracker, error) {
	if len(trackedGUIDs) == 0 {
		return nil, ErrNoTrackedGUIDs
	}
	t := &Tracker{
		buffers:     make(map[string]*damageBuffer),
		tracked:     make(map[string]struct{}, len(trackedGUIDs)),
		ordinals:    make(map[ordinalKey]int),
		logTimezone: logTimezone,
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
func NewDiagnosticTracker(logTimezone *time.Location) *Tracker {
	return &Tracker{
		allPlayers:  true,
		buffers:     make(map[string]*damageBuffer),
		ordinals:    make(map[ordinalKey]int),
		logTimezone: logTimezone,
	}
}

// Observe ingests one parsed event.
func (t *Tracker) Observe(event parser.Event) error {
	if event.Kind == parser.KindMetadata {
		t.run.observe(event, t.logTimezone)
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

	deathRes := parser.ResolveCanonicalInstant(event.Envelope.Raw, t.logTimezone)
	runCtx := t.run.context()

	hold := DeathHoldNone
	if !t.run.active || !t.run.hasCompleteStart() {
		hold = DeathHoldRunContextIncomplete
	}

	candidate := DeathCandidate{
		LineNumber:       event.LineNumber,
		Timestamp:        event.Envelope.Raw,
		DeathInstant:     deathRes.Canonical,
		DeathInstantHold: deathRes.HoldReason,
		HoldReason:       hold,
		VictimGUID:       victim,
		Run:              runCtx,
		Encounter:        t.encounter.context(),
		Causes:           rankCauses(t.damageBeforeDeath(victim, event.LineNumber)),
	}

	if hold == DeathHoldNone && deathRes.HoldReason == "" && t.run.hasCompleteStart() {
		key := ordinalKey{
			victim:       victim,
			deathInstant: deathRes.Canonical,
			runStart:     t.run.startInstant,
		}
		candidate.Ordinal = t.ordinals[key]
		candidate.OrdinalAssigned = true
		t.ordinals[key] = candidate.Ordinal + 1
	}

	t.deaths = append(t.deaths, candidate)
}

func (t *Tracker) damageBeforeDeath(victim string, lineNumber int) []DamageHit {
	if buf := t.buffers[victim]; buf != nil {
		return buf.snapshotBefore(lineNumber)
	}
	return nil
}

func (t *Tracker) tracks(guid string) bool {
	if t.allPlayers {
		return true
	}
	_, ok := t.tracked[guid]
	return ok
}
