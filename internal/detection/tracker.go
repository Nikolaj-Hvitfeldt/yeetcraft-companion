package detection

import "github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"

// Tracker consumes parsed combat-log events and accumulates death candidates.
type Tracker struct {
	tracked    map[string]struct{}
	allPlayers bool

	run       runTracker
	encounter encounterTracker
	buffers   map[string]*damageBuffer
	deaths    []DeathCandidate
}

// NewTracker returns a tracker. When trackGUIDs is empty, every player GUID
// death is recorded; otherwise only listed GUIDs match.
func NewTracker(trackGUIDs ...string) *Tracker {
	t := &Tracker{
		buffers: make(map[string]*damageBuffer),
	}
	if len(trackGUIDs) == 0 {
		t.allPlayers = true
		return t
	}
	t.tracked = make(map[string]struct{}, len(trackGUIDs))
	for _, guid := range trackGUIDs {
		if guid == "" {
			continue
		}
		t.tracked[guid] = struct{}{}
	}
	return t
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
