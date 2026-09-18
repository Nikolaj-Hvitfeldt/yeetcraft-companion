package detection

import (
	"errors"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/session"
)

// ErrIDsUnavailable is returned when a death candidate cannot produce deterministic IDs.
var ErrIDsUnavailable = errors.New("death candidate ids unavailable")

// Confidence describes how trustworthy an automated classification is.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// DeathHoldReason is a stable local hold code when a death cannot be hashed or uploaded.
type DeathHoldReason string

const (
	DeathHoldNone                 DeathHoldReason = ""
	DeathHoldRunContextIncomplete DeathHoldReason = "run_context_incomplete"
)

// SourceType is the contract-normalized damage kind for ranked causes.
type SourceType string

const (
	SourceTypeSpell         SourceType = "spell"
	SourceTypeRange         SourceType = "range"
	SourceTypeMelee         SourceType = "melee"
	SourceTypeEnvironmental SourceType = "environmental"
)

// RunContext captures the active Mythic+ run when a death occurred.
type RunContext struct {
	Active           bool
	MapID            int64
	KeystoneLevel    int64
	DungeonName      string
	StartInstant     string
	StartInstantHold parser.InstantHoldReason
}

// EncounterContext captures the active boss encounter when a death occurred.
type EncounterContext struct {
	Active        bool
	EncounterID   int64
	EncounterName string
}

// DamageHit summarizes one relevant incoming damage event during ranking.
type DamageHit struct {
	LineNumber        int
	Timestamp         string
	EventType         string
	SourceGUID        string
	SourceName        string
	SpellID           int64
	SpellName         string
	Amount            int64
	Overkill          int64
	EnvironmentalType string
}

// StorableCause is privacy-safe cause evidence suitable for persistence or upload.
type StorableCause struct {
	SourceType        SourceType
	SpellID           int64
	CreatureID        int64
	HasCreatureID     bool
	EnvironmentalType string
	Amount            int64
	Overkill          int64
	PlayerOrigin      bool
}

// CauseCandidate is one ranked explanation for a death.
type CauseCandidate struct {
	Rank       int
	Cause      StorableCause
	Confidence Confidence
}

// DeathCandidate is a tracked player death with run, encounter, and cause evidence.
type DeathCandidate struct {
	LineNumber       int
	Timestamp        string
	DeathInstant     string
	DeathInstantHold parser.InstantHoldReason
	HoldReason       DeathHoldReason
	Ordinal          int
	OrdinalAssigned  bool
	VictimGUID       string
	Run              RunContext
	Encounter        EncounterContext
	Causes           []CauseCandidate
}

// CanComputeIDs reports whether deterministic clientRunId and clientEventId values may be derived.
func (d DeathCandidate) CanComputeIDs() bool {
	if d.HoldReason != "" || d.DeathInstantHold != "" {
		return false
	}
	if d.DeathInstant == "" || d.Run.StartInstant == "" || d.Run.StartInstantHold != "" {
		return false
	}
	if !d.OrdinalAssigned {
		return false
	}
	return true
}

// ClientRunID returns the deterministic clientRunId when the candidate is hashable.
func (d DeathCandidate) ClientRunID() (string, error) {
	if !d.CanComputeIDs() {
		return "", ErrIDsUnavailable
	}
	return d.PersistClientRunID()
}

// PersistClientRunID hashes the run that was active when the death was observed.
// It does not require a hashable death instant, so a held death can still bind
// to the correct run after the session has moved on.
func (d DeathCandidate) PersistClientRunID() (string, error) {
	if d.Run.StartInstant == "" || d.Run.StartInstantHold != "" {
		return "", ErrIDsUnavailable
	}
	return session.ClientRunID(session.RunIDParams{
		ChallengeModeStartInstant: d.Run.StartInstant,
		ChallengeMapID:            int(d.Run.MapID),
		KeystoneLevel:             int(d.Run.KeystoneLevel),
	})
}

// ClientEventID returns the deterministic clientEventId when the candidate is hashable.
func (d DeathCandidate) ClientEventID() (string, error) {
	if !d.CanComputeIDs() {
		return "", ErrIDsUnavailable
	}
	runID, err := d.ClientRunID()
	if err != nil {
		return "", err
	}
	return session.ClientEventID(session.EventIDParams{
		ClientRunID:   runID,
		CharacterGUID: d.VictimGUID,
		DeathInstant:  d.DeathInstant,
		Ordinal:       d.Ordinal,
	})
}
