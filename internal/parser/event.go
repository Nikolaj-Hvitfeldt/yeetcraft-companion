package parser

import (
	"fmt"
)

type Kind int

const (
	KindEmpty Kind = iota
	KindVersionHeader
	KindMetadata
	KindCommonHeader
	KindUnknownEvent
	KindGeneric
	KindMalformed
)

type MalformedKind int

const (
	MalformedNone MalformedKind = iota
	MalformedCSV
	MalformedVersionHeader
	MalformedCommonHeader
	MalformedOther
)

type CommonHeader struct {
	EventType       string
	SourceGUID      string
	SourceName      string
	SourceNameIsNil bool
	SourceFlags     *uint32
	SourceRaidFlags *uint32
	DestGUID        string
	DestName        string
	DestNameIsNil   bool
	DestFlags       *uint32
	DestRaidFlags   *uint32
}

type Event struct {
	LineNumber int
	Envelope   Envelope
	Payload    string
	Fields     []string
	Kind       Kind
	EventType  string
	Version    *VersionHeader
	Common     *CommonHeader
	Structural *CommonHeader
	Warnings   []string
	Err        error
	Malformed  MalformedKind
	Typed      TypedResult
}

var metadataEvents = map[string]struct{}{
	"ENCOUNTER_START":      {},
	"ENCOUNTER_END":        {},
	"CHALLENGE_MODE_START": {},
	"CHALLENGE_MODE_END":   {},
}

// commonHeaderEvents is deliberately explicit. Field count alone never makes
// an event type recognized, and no wildcard/prefix matching is used.
var commonHeaderEvents = map[string]struct{}{
	"SPELL_DAMAGE":         {},
	"RANGE_DAMAGE":         {},
	"SWING_DAMAGE":         {},
	"ENVIRONMENTAL_DAMAGE": {},
	"UNIT_DIED":            {},
}

func extractCommonHeader(fields []string) (*CommonHeader, []string, error) {
	if len(fields) < 9 {
		return nil, nil, fmt.Errorf("common header requires 9 fields, got %d", len(fields))
	}

	header := &CommonHeader{
		EventType:       fields[0],
		SourceGUID:      fields[1],
		SourceName:      fields[2],
		SourceNameIsNil: fields[2] == "nil",
		DestGUID:        fields[5],
		DestName:        fields[6],
		DestNameIsNil:   fields[6] == "nil",
	}
	if header.SourceNameIsNil {
		header.SourceName = ""
	}
	if header.DestNameIsNil {
		header.DestName = ""
	}

	var warnings []string
	header.SourceFlags = parseFlag(fields[3], "source flags", &warnings)
	header.SourceRaidFlags = parseFlag(fields[4], "source raid flags", &warnings)
	header.DestFlags = parseFlag(fields[7], "destination flags", &warnings)
	header.DestRaidFlags = parseFlag(fields[8], "destination raid flags", &warnings)
	return header, warnings, nil
}

func parseFlag(raw, label string, warnings *[]string) *uint32 {
	value, ok := parseHexUint32(raw)
	if !ok {
		*warnings = append(*warnings, fmt.Sprintf("invalid %s", label))
		return nil
	}
	return &value
}
