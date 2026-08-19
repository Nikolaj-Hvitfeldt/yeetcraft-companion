package detection

// Confidence describes how trustworthy an automated classification is.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// RunContext captures the active Mythic+ run when a death occurred.
type RunContext struct {
	Active        bool
	MapID         int64
	KeystoneLevel int64
	DungeonName   string
}

// EncounterContext captures the active boss encounter when a death occurred.
type EncounterContext struct {
	Active        bool
	EncounterID   int64
	EncounterName string
}

// DamageHit summarizes one relevant incoming damage event.
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

// CauseCandidate is one ranked explanation for a death.
type CauseCandidate struct {
	Rank       int
	Hit        DamageHit
	Confidence Confidence
}

// DeathCandidate is a tracked player death with run, encounter, and cause evidence.
type DeathCandidate struct {
	LineNumber int
	Timestamp  string
	VictimGUID string
	Run        RunContext
	Encounter  EncounterContext
	Causes     []CauseCandidate
}
