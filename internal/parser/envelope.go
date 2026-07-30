package parser

import (
	"regexp"
	"strings"
	"time"
)

type EnvelopeKind int

const (
	EnvelopeNone EnvelopeKind = iota
	EnvelopeReferenceShape
	EnvelopeObservedFallback
)

type Envelope struct {
	Kind   EnvelopeKind
	Raw    string
	Parsed bool
}

type EnvelopeSplit struct {
	Envelope Envelope
	Payload  string
}

const provisionalTimestampPattern = `(\d{1,2}/\d{1,2}/\d{4} \d{1,2}:\d{2}:\d{2}\.\d+(?:[+-]\d{2}(?::?\d{2})?)?)`

var (
	referenceEnvelope = regexp.MustCompile(`^` + provisionalTimestampPattern + `(?:\t| {2,})(.*)$`)
	observedEnvelope  = regexp.MustCompile(`^` + provisionalTimestampPattern + ` (.*)$`)
)

// SplitEnvelope conservatively separates the provisional timestamp envelope
// from CSV. Neither supported envelope shape is verified against a real 12.0+
// log, so this function is intentionally isolated and replaceable.
func SplitEnvelope(rawLine string) EnvelopeSplit {
	if strings.HasPrefix(rawLine, "COMBAT_LOG_VERSION,") {
		return EnvelopeSplit{Envelope: Envelope{Kind: EnvelopeNone}, Payload: rawLine}
	}
	if match := referenceEnvelope.FindStringSubmatch(rawLine); match != nil {
		_, parsed := TryParseEnvelopeTimestamp(match[1])
		return EnvelopeSplit{
			Envelope: Envelope{Kind: EnvelopeReferenceShape, Raw: match[1], Parsed: parsed},
			Payload:  match[2],
		}
	}
	if match := observedEnvelope.FindStringSubmatch(rawLine); match != nil {
		_, parsed := TryParseEnvelopeTimestamp(match[1])
		return EnvelopeSplit{
			Envelope: Envelope{Kind: EnvelopeObservedFallback, Raw: match[1], Parsed: parsed},
			Payload:  match[2],
		}
	}
	return EnvelopeSplit{Envelope: Envelope{Kind: EnvelopeNone}, Payload: rawLine}
}

func TryParseEnvelopeTimestamp(raw string) (time.Time, bool) {
	layouts := []string{
		"1/2/2006 15:04:05.999999999Z07:00",
		"1/2/2006 15:04:05.999999999Z0700",
		"1/2/2006 15:04:05.999999999Z07",
		"1/2/2006 15:04:05.999999999",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
