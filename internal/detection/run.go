package detection

import (
	"time"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

type runTracker struct {
	active           bool
	mapID            int64
	keystoneLevel    int64
	dungeonName      string
	startInstant     string
	startInstantHold parser.InstantHoldReason
}

func (r *runTracker) context() RunContext {
	return RunContext{
		Active:           r.active,
		MapID:            r.mapID,
		KeystoneLevel:    r.keystoneLevel,
		DungeonName:      r.dungeonName,
		StartInstant:     r.startInstant,
		StartInstantHold: r.startInstantHold,
	}
}

func (r *runTracker) hasCompleteStart() bool {
	return r.startInstant != "" && r.startInstantHold == ""
}

func (r *runTracker) observe(event parser.Event, loc *time.Location) {
	switch event.EventType {
	case "CHALLENGE_MODE_START":
		r.observeStart(event, loc)
	case "CHALLENGE_MODE_END":
		r.observeEnd(event)
	}
}

func (r *runTracker) observeStart(event parser.Event, loc *time.Location) {
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
	r.active = true
	r.mapID = mapID
	r.dungeonName = fields[1]
	r.keystoneLevel = keyLevel
	r.startInstant = res.Canonical
	r.startInstantHold = res.HoldReason
}

func (r *runTracker) observeEnd(event parser.Event) {
	if event.Typed.Status != parser.TypedStatusParsed {
		return
	}
	payload, ok := event.Typed.Payload.(parser.ChallengeModeEndPayload)
	if !ok {
		return
	}
	// Retail logs emit zeroed END lines when entering an instance before START.
	if !payload.Success && payload.KeystoneLevel == 0 {
		return
	}
	if payload.Success {
		r.active = false
	}
}

type encounterTracker struct {
	active        bool
	encounterID   int64
	encounterName string
}

func (e *encounterTracker) context() EncounterContext {
	if !e.active {
		return EncounterContext{}
	}
	return EncounterContext{
		Active:        e.active,
		EncounterID:   e.encounterID,
		EncounterName: e.encounterName,
	}
}

func (e *encounterTracker) observe(event parser.Event) {
	switch event.EventType {
	case "ENCOUNTER_START":
		if event.Typed.Status != parser.TypedStatusParsed {
			return
		}
		payload, ok := event.Typed.Payload.(parser.EncounterStartPayload)
		if !ok {
			return
		}
		e.active = true
		e.encounterID = payload.EncounterID
		e.encounterName = payload.EncounterName
	case "ENCOUNTER_END":
		if event.Typed.Status != parser.TypedStatusParsed {
			return
		}
		payload, ok := event.Typed.Payload.(parser.EncounterEndPayload)
		if !ok {
			return
		}
		if e.active && e.encounterID == payload.EncounterID {
			e.active = false
			e.encounterID = 0
			e.encounterName = ""
		}
	}
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
