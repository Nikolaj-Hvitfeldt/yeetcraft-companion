package parser

import "fmt"

type TypedStatus uint8

const (
	TypedStatusNotApplicable TypedStatus = iota
	TypedStatusParsed
	TypedStatusInvalid
)

type TypedPayload interface {
	typedPayload()
}

type TypedResult struct {
	Status      TypedStatus
	Payload     TypedPayload
	Error       *TypedPayloadError
	Diagnostics ValidationDiagnostics
}

type TypedErrorKind uint8

const (
	TypedErrorNone TypedErrorKind = iota
	TypedErrorFieldCount
	TypedErrorEmptyRequired
	TypedErrorInteger
	TypedErrorHex
	TypedErrorFloat
	TypedErrorBoolean
)

type TypedPayloadError struct {
	Kind        TypedErrorKind
	EventType   string
	FieldName   string
	FieldIndex  int
	Expected    int
	ExpectedMax int
	Actual      int
}

func (e *TypedPayloadError) Error() string {
	if e == nil {
		return ""
	}
	if e.Kind == TypedErrorFieldCount {
		if e.ExpectedMax > e.Expected {
			return fmt.Sprintf("parse %s typed payload: unexpected field count (expected %d-%d, got %d)", e.EventType, e.Expected, e.ExpectedMax, e.Actual)
		}
		return fmt.Sprintf("parse %s typed payload: unexpected field count (expected %d, got %d)", e.EventType, e.Expected, e.Actual)
	}
	return fmt.Sprintf("parse %s typed payload field %d (%s): invalid %s", e.EventType, e.FieldIndex, e.FieldName, e.Kind)
}

func (k TypedErrorKind) String() string {
	switch k {
	case TypedErrorFieldCount:
		return "field count"
	case TypedErrorEmptyRequired:
		return "required value"
	case TypedErrorInteger:
		return "integer"
	case TypedErrorHex:
		return "hexadecimal value"
	case TypedErrorFloat:
		return "float"
	case TypedErrorBoolean:
		return "boolean"
	default:
		return "typed value"
	}
}

type ValidationDiagnostics uint32

const (
	DiagnosticAdvancedInfoGUIDMismatch ValidationDiagnostics = 1 << iota
	DiagnosticEnvironmentalSourceNotZero
	DiagnosticSwingSchoolUnexpected
	DiagnosticAbilityHintUnknown
	DiagnosticEnvironmentalTypeUnknown
	DiagnosticAdvancedUnknownFieldNonZero
)

func (d ValidationDiagnostics) Has(flag ValidationDiagnostics) bool {
	return d&flag != 0
}

type BoolOrNil uint8

const (
	BoolNil BoolOrNil = iota
	BoolTrue
)

type OptionalBoolOrNil uint8

const (
	OptionalBoolOmitted OptionalBoolOrNil = iota
	OptionalBoolNil
	OptionalBoolTrue
)

type AbilityHintKind uint8

const (
	AbilityHintUnknown AbilityHintKind = iota
	AbilityHintSingleTarget
	AbilityHintAreaOfEffect
)

type AbilityHint struct {
	Value string
	Kind  AbilityHintKind
}

type EnvironmentalTypeKind uint8

const (
	EnvironmentalTypeUnknown EnvironmentalTypeKind = iota
	EnvironmentalTypeFalling
	EnvironmentalTypeLava
	EnvironmentalTypeFire
	EnvironmentalTypeSlime
	EnvironmentalTypeDrowning
	EnvironmentalTypeFatigue
)

type EnvironmentalType struct {
	Value string
	Kind  EnvironmentalTypeKind
}

type SpellPrefix struct {
	ID     int64
	Name   string
	School uint32
}

type AdvancedCombatLog struct {
	InfoGUID     string
	OwnerGUID    string
	CurrentHP    int64
	MaxHP        int64
	AttackPower  int64
	SpellPower   int64
	Armor        int64
	Absorb       int64
	Unknown1     int64
	Unknown2     int64
	PowerType    int64
	CurrentPower int64
	MaxPower     int64
	PowerCost    int64
	PositionX    float64
	PositionY    float64
	UIMapID      int64
	Facing       float64
	ItemLevel    int64
}

type DamageSuffix struct {
	BaseAmount int64
	RawAmount  int64
	Overkill   int64
	School     uint32
	Resisted   int64
	Blocked    int64
	Absorbed   int64
	Critical   BoolOrNil
	Glancing   BoolOrNil
	Crushing   BoolOrNil
}

type SpellDamagePayload struct {
	Spell       SpellPrefix
	Target      AdvancedCombatLog
	Damage      DamageSuffix
	AbilityHint AbilityHint
}

func (SpellDamagePayload) typedPayload() {}

type RangeDamagePayload struct {
	Spell       SpellPrefix
	Target      AdvancedCombatLog
	Damage      DamageSuffix
	AbilityHint AbilityHint
}

func (RangeDamagePayload) typedPayload() {}

type SwingDamagePayload struct {
	Source    AdvancedCombatLog
	Damage    DamageSuffix
	IsOffHand OptionalBoolOrNil
}

func (SwingDamagePayload) typedPayload() {}

type EnvironmentalDamagePayload struct {
	Target            AdvancedCombatLog
	EnvironmentalType EnvironmentalType
	Damage            DamageSuffix
}

func (EnvironmentalDamagePayload) typedPayload() {}

type EncounterStartPayload struct {
	EncounterID   int64
	EncounterName string
	DifficultyID  int64
	GroupSize     int64
	InstanceID    *int64
}

func (EncounterStartPayload) typedPayload() {}

type EncounterEndPayload struct {
	EncounterID   int64
	EncounterName string
	DifficultyID  int64
	GroupSize     int64
	Success       bool
	DurationMS    *int64
}

func (EncounterEndPayload) typedPayload() {}

type ChallengeModeEndPayload struct {
	MapID             int64
	Success           bool
	KeystoneLevel     int64
	TotalTimeMS       *int64
	OnTimeSeconds     *float64
	TimerLimitSeconds *int64
}

func (ChallengeModeEndPayload) typedPayload() {}
