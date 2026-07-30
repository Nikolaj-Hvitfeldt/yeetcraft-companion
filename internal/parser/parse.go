package parser

import (
	"errors"
	"fmt"
	"strings"
)

// ParseLine parses one complete line and updates version state.
func ParseLine(lineNumber int, rawLine string, state *ParserState) Event {
	if state == nil {
		state = &ParserState{}
	}
	event := Event{LineNumber: lineNumber}
	if rawLine == "" {
		event.Kind = KindEmpty
		return event
	}

	split := SplitEnvelope(rawLine)
	event.Envelope = split.Envelope
	event.Payload = split.Payload

	fields, err := TokenizeCSV(split.Payload)
	if err != nil {
		event.Kind = KindMalformed
		if split.Payload == "COMBAT_LOG_VERSION" || strings.HasPrefix(split.Payload, "COMBAT_LOG_VERSION,") {
			state.applyVersion(VersionHeader{}, err)
			event.EventType = "COMBAT_LOG_VERSION"
			event.Malformed = MalformedVersionHeader
			event.Err = &LineError{Op: "parse version", LineNumber: lineNumber, Err: err}
		} else {
			event.Malformed = MalformedCSV
			event.Err = &LineError{Op: "tokenize", LineNumber: lineNumber, Err: err}
		}
		return event
	}
	event.Fields = fields
	if len(fields) == 0 || fields[0] == "" {
		event.Kind = KindMalformed
		event.Malformed = MalformedOther
		event.Err = &LineError{Op: "parse", LineNumber: lineNumber, Err: fmt.Errorf("missing event type")}
		return event
	}
	event.EventType = fields[0]

	if IsVersionHeader(fields) {
		header, versionErr := ParseVersionHeader(fields)
		state.applyVersion(header, versionErr)
		isUnsupported := errors.Is(versionErr, ErrUnsupportedFormatVersion) ||
			errors.Is(versionErr, ErrUnsupportedProject)
		if versionErr != nil && !isUnsupported {
			event.Kind = KindMalformed
			event.Malformed = MalformedVersionHeader
			event.Err = &LineError{Op: "parse version", LineNumber: lineNumber, Err: versionErr}
			return event
		}
		event.Kind = KindVersionHeader
		event.Version = &header
		if versionErr != nil {
			event.Err = &LineError{Op: "parse version", LineNumber: lineNumber, Err: versionErr}
		}
		return event
	}

	if state.Format == FormatStateQuarantinedUnsupported || state.Format == FormatStateNone {
		event.Kind = KindGeneric
		event.Warnings = append(event.Warnings, "record preserved without supported format interpretation")
		return event
	}

	if _, ok := metadataEvents[event.EventType]; ok {
		event.Kind = KindMetadata
		return event
	}

	if _, ok := commonHeaderEvents[event.EventType]; ok {
		common, warnings, commonErr := extractCommonHeader(fields)
		if commonErr != nil {
			event.Kind = KindMalformed
			event.Malformed = MalformedCommonHeader
			event.Err = &LineError{Op: "parse common header", LineNumber: lineNumber, Err: commonErr}
			return event
		}
		event.Kind = KindCommonHeader
		event.Common = common
		event.Warnings = append(event.Warnings, warnings...)
		return event
	}

	event.Kind = KindUnknownEvent
	if len(fields) >= 9 {
		structural, warnings, commonErr := extractCommonHeader(fields)
		if commonErr == nil {
			event.Structural = structural
			event.Warnings = append(event.Warnings, warnings...)
		}
	}
	return event
}
