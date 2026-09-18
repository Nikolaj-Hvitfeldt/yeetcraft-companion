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

// InstantHoldReason is a stable local hold code when canonical instant resolution
// cannot produce a hash-safe RFC 3339 UTC instant.
type InstantHoldReason string

const (
	InstantHoldNone               InstantHoldReason = ""
	InstantHoldMissingLogTimezone InstantHoldReason = "missing_log_timezone"
	InstantHoldDstAmbiguous       InstantHoldReason = "dst_ambiguous"
	InstantHoldInvalidInstant     InstantHoldReason = "invalid_instant"
)

// InstantResolution is the result of ResolveCanonicalInstant.
type InstantResolution struct {
	Canonical  string
	HoldReason InstantHoldReason
}

const canonicalInstantLayout = "2006-01-02T15:04:05.000000000Z"

var envelopeComponentsPattern = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{4}) (\d{1,2}):(\d{2}):(\d{2})(?:\.(\d+))?(.*)$`)

// ResolveCanonicalInstant converts a provisional combat-log envelope timestamp into
// the companion v1 canonical RFC 3339 UTC instant. Offset-bearing stamps convert
// directly to UTC. Timezone-less stamps require a non-nil location. Hold reasons
// are returned instead of errors so scanning can continue.
func ResolveCanonicalInstant(raw string, loc *time.Location) InstantResolution {
	if raw == "" {
		return InstantResolution{HoldReason: InstantHoldInvalidInstant}
	}

	if suffix, ok := extractTimezoneSuffix(raw); ok {
		if !isValidTimezoneOffset(suffix) {
			return InstantResolution{HoldReason: InstantHoldInvalidInstant}
		}
		components, ok := parseEnvelopeComponents(raw)
		if !ok || !validEnvelopeDate(components) {
			return InstantResolution{HoldReason: InstantHoldInvalidInstant}
		}
		parsed, ok := TryParseEnvelopeTimestamp(raw)
		if !ok || !offsetWallClockMatches(parsed, components, suffix) {
			return InstantResolution{HoldReason: InstantHoldInvalidInstant}
		}
		return InstantResolution{Canonical: formatCanonicalInstant(parsed.UTC())}
	}

	if loc == nil {
		return InstantResolution{HoldReason: InstantHoldMissingLogTimezone}
	}

	components, ok := parseEnvelopeComponents(raw)
	if !ok {
		return InstantResolution{HoldReason: InstantHoldInvalidInstant}
	}
	if !validEnvelopeDate(components) {
		return InstantResolution{HoldReason: InstantHoldInvalidInstant}
	}

	local, exists := localWallTimeFromComponents(components, loc)
	if !exists {
		return InstantResolution{HoldReason: InstantHoldDstAmbiguous}
	}
	if hasDuplicateWallTime(loc, local) {
		return InstantResolution{HoldReason: InstantHoldDstAmbiguous}
	}

	return InstantResolution{Canonical: formatCanonicalInstant(local.UTC())}
}

type envelopeComponents struct {
	month int
	day   int
	year  int
	hour  int
	min   int
	sec   int
	nsec  int
}

func parseEnvelopeComponents(raw string) (envelopeComponents, bool) {
	match := envelopeComponentsPattern.FindStringSubmatch(raw)
	if match == nil {
		return envelopeComponents{}, false
	}

	month, err := strconv.Atoi(match[1])
	if err != nil || month < 1 || month > 12 {
		return envelopeComponents{}, false
	}
	day, err := strconv.Atoi(match[2])
	if err != nil || day < 1 || day > 31 {
		return envelopeComponents{}, false
	}
	year, err := strconv.Atoi(match[3])
	if err != nil {
		return envelopeComponents{}, false
	}
	hour, err := strconv.Atoi(match[4])
	if err != nil || hour < 0 || hour > 23 {
		return envelopeComponents{}, false
	}
	min, err := strconv.Atoi(match[5])
	if err != nil || min < 0 || min > 59 {
		return envelopeComponents{}, false
	}
	sec, err := strconv.Atoi(match[6])
	if err != nil || sec < 0 || sec > 59 {
		return envelopeComponents{}, false
	}

	nsec := 0
	if match[7] != "" {
		var ok bool
		nsec, ok = fractionDigitsToNanoseconds(match[7])
		if !ok {
			return envelopeComponents{}, false
		}
	}

	suffix := match[8]
	if suffix != "" {
		if suffix[0] != '+' && suffix[0] != '-' {
			return envelopeComponents{}, false
		}
		if !isValidTimezoneOffset(suffix) {
			return envelopeComponents{}, false
		}
	}

	return envelopeComponents{
		month: month,
		day:   day,
		year:  year,
		hour:  hour,
		min:   min,
		sec:   sec,
		nsec:  nsec,
	}, true
}

func fractionDigitsToNanoseconds(frac string) (int, bool) {
	if frac == "" {
		return 0, false
	}
	for i := 0; i < len(frac); i++ {
		if frac[i] < '0' || frac[i] > '9' {
			return 0, false
		}
	}
	if len(frac) > 9 {
		frac = frac[:9]
	}
	padded := frac + strings.Repeat("0", 9-len(frac))
	nsec, err := strconv.Atoi(padded)
	if err != nil {
		return 0, false
	}
	return nsec, true
}

func validEnvelopeDate(c envelopeComponents) bool {
	if c.month < 1 || c.month > 12 || c.day < 1 || c.day > 31 {
		return false
	}
	if time.Date(c.year, time.Month(c.month), c.day, 0, 0, 0, 0, time.UTC).Month() != time.Month(c.month) {
		return false
	}
	if time.Date(c.year, time.Month(c.month), c.day, 0, 0, 0, 0, time.UTC).Day() != c.day {
		return false
	}
	return true
}

func offsetWallClockMatches(parsed time.Time, c envelopeComponents, suffix string) bool {
	offsetSec, ok := timezoneOffsetSeconds(suffix)
	if !ok {
		return false
	}
	loc := time.FixedZone("envelope-offset", offsetSec)
	local := parsed.In(loc)
	y, m, d := local.Date()
	h, min, sec := local.Clock()
	return y == c.year &&
		int(m) == c.month &&
		d == c.day &&
		h == c.hour &&
		min == c.min &&
		sec == c.sec &&
		local.Nanosecond() == c.nsec
}

func timezoneOffsetSeconds(suffix string) (int, bool) {
	if !isValidTimezoneOffset(suffix) {
		return 0, false
	}

	sign := 1
	body := suffix[1:]
	if suffix[0] == '-' {
		sign = -1
	}

	var hour, minute int
	switch {
	case strings.Contains(body, ":"):
		hourPart, minutePart, ok := strings.Cut(body, ":")
		if !ok || len(hourPart) != 2 || len(minutePart) != 2 {
			return 0, false
		}
		var err error
		if hour, err = strconv.Atoi(hourPart); err != nil {
			return 0, false
		}
		if minute, err = strconv.Atoi(minutePart); err != nil {
			return 0, false
		}
	case len(body) == 4:
		var err error
		if hour, err = strconv.Atoi(body[:2]); err != nil {
			return 0, false
		}
		if minute, err = strconv.Atoi(body[2:]); err != nil {
			return 0, false
		}
	case len(body) == 2:
		var err error
		if hour, err = strconv.Atoi(body); err != nil {
			return 0, false
		}
	default:
		return 0, false
	}

	return sign * ((hour * 60 * 60) + (minute * 60)), true
}

func localWallTimeFromComponents(c envelopeComponents, loc *time.Location) (time.Time, bool) {
	t := time.Date(c.year, time.Month(c.month), c.day, c.hour, c.min, c.sec, c.nsec, loc)
	y, m, d := t.In(loc).Date()
	h, min, sec := t.In(loc).Clock()
	if y != c.year || int(m) != c.month || d != c.day || h != c.hour || min != c.min || sec != c.sec {
		return time.Time{}, false
	}
	if t.In(loc).Nanosecond() != c.nsec {
		return time.Time{}, false
	}
	return t, true
}

func hasDuplicateWallTime(loc *time.Location, t time.Time) bool {
	local := t.In(loc)
	y, m, d := local.Date()
	h, min, sec := local.Clock()
	ns := local.Nanosecond()

	start := t.UTC().Add(-time.Hour)
	for offset := time.Duration(0); offset <= 2*time.Hour; offset += time.Second {
		candidate := start.Add(offset)
		cy, cm, cd := candidate.In(loc).Date()
		ch, cmin, cs := candidate.In(loc).Clock()
		cns := candidate.In(loc).Nanosecond()
		if cy == y && int(cm) == int(m) && cd == d && ch == h && cmin == min && cs == sec && cns == ns {
			if !candidate.Equal(t.UTC()) {
				return true
			}
		}
	}
	return false
}

func formatCanonicalInstant(t time.Time) string {
	return t.UTC().Format(canonicalInstantLayout)
}
