package parser

import (
	"regexp"
	"strconv"
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
	if suffix, ok := extractTimezoneSuffix(raw); ok && !isValidTimezoneOffset(suffix) {
		return time.Time{}, false
	}

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

// extractTimezoneSuffix returns a signed numeric offset suffix that follows
// the fractional-second digits, if present.
func extractTimezoneSuffix(raw string) (string, bool) {
	dot := strings.LastIndex(raw, ".")
	if dot < 0 || dot == len(raw)-1 {
		return "", false
	}
	rest := raw[dot+1:]
	i := 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		i++
	}
	if i >= len(rest) {
		return "", false
	}
	suffix := rest[i:]
	if suffix[0] != '+' && suffix[0] != '-' {
		return "", false
	}
	return suffix, true
}

// isValidTimezoneOffset validates signed offset syntax independently of Go's
// time.Parse behavior, which varies across Go versions for out-of-range values.
func isValidTimezoneOffset(suffix string) bool {
	if len(suffix) < 3 || (suffix[0] != '+' && suffix[0] != '-') {
		return false
	}

	body := suffix[1:]
	var hour, minute int
	hasMinutes := false

	switch {
	case strings.Contains(body, ":"):
		hourPart, minutePart, ok := strings.Cut(body, ":")
		if !ok || len(hourPart) != 2 || len(minutePart) != 2 {
			return false
		}
		var err error
		if hour, err = strconv.Atoi(hourPart); err != nil {
			return false
		}
		if minute, err = strconv.Atoi(minutePart); err != nil {
			return false
		}
		hasMinutes = true
	case len(body) == 4:
		var err error
		if hour, err = strconv.Atoi(body[:2]); err != nil {
			return false
		}
		if minute, err = strconv.Atoi(body[2:]); err != nil {
			return false
		}
		hasMinutes = true
	case len(body) == 2:
		var err error
		if hour, err = strconv.Atoi(body); err != nil {
			return false
		}
	default:
		return false
	}

	if hour < 0 || hour > 23 {
		return false
	}
	if hasMinutes && (minute < 0 || minute > 59) {
		return false
	}
	return true
}
