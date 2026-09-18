package parser

import (
	"os"
	"strings"
	"testing"
	"time"

	_ "time/tzdata"
)

func TestSplitEnvelopePrecedence(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantKind    EnvelopeKind
		wantRaw     string
		wantPayload string
	}{
		{
			name:        "unprefixed version",
			line:        "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1",
			wantKind:    EnvelopeNone,
			wantPayload: "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1",
		},
		{
			name:        "reference two spaces with signed offset",
			line:        "1/15/2026 20:00:01.0000+02:00  SPELL_DAMAGE,a",
			wantKind:    EnvelopeReferenceShape,
			wantRaw:     "1/15/2026 20:00:01.0000+02:00",
			wantPayload: "SPELL_DAMAGE,a",
		},
		{
			name:        "reference tab with signed offset",
			line:        "1/15/2026 20:00:01.0000-0500\tSPELL_DAMAGE,a",
			wantKind:    EnvelopeReferenceShape,
			wantRaw:     "1/15/2026 20:00:01.0000-0500",
			wantPayload: "SPELL_DAMAGE,a",
		},
		{
			name:        "observed one space",
			line:        "1/15/2026 20:00:01.0000 SPELL_DAMAGE,a",
			wantKind:    EnvelopeObservedFallback,
			wantRaw:     "1/15/2026 20:00:01.0000",
			wantPayload: "SPELL_DAMAGE,a",
		},
		{
			name:        "ambiguous remains unmodified",
			line:        "not a timestamp SPELL_DAMAGE,a",
			wantKind:    EnvelopeNone,
			wantPayload: "not a timestamp SPELL_DAMAGE,a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitEnvelope(tt.line)
			if got.Envelope.Kind != tt.wantKind || got.Envelope.Raw != tt.wantRaw || got.Payload != tt.wantPayload {
				t.Fatalf("SplitEnvelope() = %#v, want kind=%v raw=%q payload=%q", got, tt.wantKind, tt.wantRaw, tt.wantPayload)
			}
		})
	}
}

func TestTryParseEnvelopeTimestampIsOptional(t *testing.T) {
	if _, ok := TryParseEnvelopeTimestamp("1/15/2026 20:00:01.0000"); !ok {
		t.Fatal("expected provisional timestamp to parse")
	}
	if _, ok := TryParseEnvelopeTimestamp("unresolved-shape"); ok {
		t.Fatal("unexpected parse success")
	}

	split := SplitEnvelope("1/15/2026 20:00:01.0000+24:99  SYNTHETIC_UNKNOWN_EVENT,a")
	if split.Envelope.Kind != EnvelopeReferenceShape || split.Envelope.Parsed {
		t.Fatalf("split = %#v, want matched but unparsed reference shape", split)
	}
	fields, err := TokenizeCSV(split.Payload)
	if err != nil || len(fields) != 2 || fields[0] != "SYNTHETIC_UNKNOWN_EVENT" {
		t.Fatalf("payload tokenization fields=%#v err=%v", fields, err)
	}
}

func TestTryParseEnvelopeTimestampTimezoneOffsets(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantParsed bool
	}{
		{
			name:       "no timezone suffix",
			raw:        "1/15/2026 20:00:01.0000",
			wantParsed: true,
		},
		{
			name:       "zero offset with colon",
			raw:        "1/15/2026 20:00:01.0000+00:00",
			wantParsed: true,
		},
		{
			name:       "positive offset with colon",
			raw:        "1/15/2026 20:00:01.0000+02:00",
			wantParsed: true,
		},
		{
			name:       "negative offset without colon",
			raw:        "1/15/2026 20:00:01.0000-0500",
			wantParsed: true,
		},
		{
			name:       "maximum positive offset with colon",
			raw:        "1/15/2026 20:00:01.0000+23:59",
			wantParsed: true,
		},
		{
			name:       "maximum negative offset without colon",
			raw:        "1/15/2026 20:00:01.0000-2359",
			wantParsed: true,
		},
		{
			name:       "hour-only positive offset",
			raw:        "1/15/2026 20:00:01.0000+07",
			wantParsed: true,
		},
		{
			name:       "hour out of range",
			raw:        "1/15/2026 20:00:01.0000+24:00",
			wantParsed: false,
		},
		{
			name:       "hour and minute out of range",
			raw:        "1/15/2026 20:00:01.0000+24:99",
			wantParsed: false,
		},
		{
			name:       "minute out of range",
			raw:        "1/15/2026 20:00:01.0000+12:60",
			wantParsed: false,
		},
		{
			name:       "incomplete offset missing minutes",
			raw:        "1/15/2026 20:00:01.0000+02:",
			wantParsed: false,
		},
		{
			name:       "incomplete offset single minute digit",
			raw:        "1/15/2026 20:00:01.0000+02:0",
			wantParsed: false,
		},
		{
			name:       "incomplete offset three digit body",
			raw:        "1/15/2026 20:00:01.0000+024",
			wantParsed: false,
		},
		{
			name:       "incomplete offset sign only",
			raw:        "1/15/2026 20:00:01.0000+",
			wantParsed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotParsed := TryParseEnvelopeTimestamp(tt.raw)
			if gotParsed != tt.wantParsed {
				t.Fatalf("TryParseEnvelopeTimestamp(%q) parsed = %v, want %v", tt.raw, gotParsed, tt.wantParsed)
			}
		})
	}
}

func TestResolveCanonicalInstant(t *testing.T) {
	utc := time.UTC
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation(America/New_York): %v", err)
	}

	tests := []struct {
		name      string
		raw       string
		loc       *time.Location
		wantCanon string
		wantHold  InstantHoldReason
	}{
		{
			name:      "offset-bearing with colon offset",
			raw:       "1/15/2026 20:00:01.0000+02:00",
			wantCanon: "2026-01-15T18:00:01.000000000Z",
		},
		{
			name:      "offset-bearing without colon offset",
			raw:       "1/15/2026 20:00:01.0000-0500",
			wantCanon: "2026-01-16T01:00:01.000000000Z",
		},
		{
			name:      "offset-bearing hour-only offset",
			raw:       "1/15/2026 20:00:01.0000+07",
			wantCanon: "2026-01-15T13:00:01.000000000Z",
		},
		{
			name:      "timezone-less with UTC location",
			raw:       "1/15/2026 20:00:01.123",
			loc:       utc,
			wantCanon: "2026-01-15T20:00:01.123000000Z",
		},
		{
			name:     "timezone-less without location",
			raw:      "1/15/2026 20:00:01.0000",
			wantHold: InstantHoldMissingLogTimezone,
		},
		{
			name:     "spring-forward nonexistent local time",
			raw:      "3/9/2025 2:30:00.0",
			loc:      newYork,
			wantHold: InstantHoldDstAmbiguous,
		},
		{
			name:     "fall-back ambiguous local time",
			raw:      "11/2/2025 1:30:00.0",
			loc:      newYork,
			wantHold: InstantHoldDstAmbiguous,
		},
		{
			name:     "invalid calendar date",
			raw:      "2/31/2026 12:00:00.0",
			loc:      utc,
			wantHold: InstantHoldInvalidInstant,
		},
		{
			name:     "invalid offset-bearing stamp",
			raw:      "1/15/2026 20:00:01.0000+24:00",
			wantHold: InstantHoldInvalidInstant,
		},
		{
			name:      "one-digit fractional padding",
			raw:       "1/15/2026 20:00:01.1+02:00",
			wantCanon: "2026-01-15T18:00:01.100000000Z",
		},
		{
			name:      "three-digit fractional padding",
			raw:       "1/15/2026 20:00:01.123+02:00",
			wantCanon: "2026-01-15T18:00:01.123000000Z",
		},
		{
			name:      "six-digit fractional padding",
			raw:       "1/15/2026 20:00:01.123456+02:00",
			wantCanon: "2026-01-15T18:00:01.123456000Z",
		},
		{
			name:      "nine-digit fractional preserved",
			raw:       "1/15/2026 20:00:01.123456789+02:00",
			wantCanon: "2026-01-15T18:00:01.123456789Z",
		},
		{
			name:     "empty stamp",
			raw:      "",
			wantHold: InstantHoldInvalidInstant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCanonicalInstant(tt.raw, tt.loc)
			if got.HoldReason != tt.wantHold {
				t.Fatalf("HoldReason = %q, want %q (canonical=%q)", got.HoldReason, tt.wantHold, got.Canonical)
			}
			if got.Canonical != tt.wantCanon {
				t.Fatalf("Canonical = %q, want %q", got.Canonical, tt.wantCanon)
			}
			if tt.wantCanon != "" {
				if !strings.HasSuffix(got.Canonical, "Z") {
					t.Fatalf("canonical instant must end with Z: %q", got.Canonical)
				}
				if len(got.Canonical) != len("2026-01-15T18:00:01.123456789Z") {
					t.Fatalf("canonical instant length = %d, want fixed RFC3339 nanosecond form", len(got.Canonical))
				}
				if _, err := time.Parse("2006-01-02T15:04:05.000000000Z", got.Canonical); err != nil {
					t.Fatalf("canonical instant is not parseable: %v", err)
				}
			}
			if tt.wantHold == InstantHoldNone && got.Canonical == "" {
				t.Fatal("expected canonical instant without hold reason")
			}
			if tt.wantHold != InstantHoldNone && got.Canonical != "" {
				t.Fatalf("hold reason %q must not produce canonical instant %q", got.HoldReason, got.Canonical)
			}
		})
	}
}

func TestResolveCanonicalInstantSyntheticFixtures(t *testing.T) {
	utc := time.UTC
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation(America/New_York): %v", err)
	}

	tests := []struct {
		name      string
		fixture   string
		loc       *time.Location
		wantCanon string
		wantHold  InstantHoldReason
	}{
		{
			name:      "timezone-less fixture with UTC",
			fixture:   "../../testdata/logs/synthetic/timestamp-timezone-less.txt",
			loc:       utc,
			wantCanon: "2026-01-15T20:16:00.000000000Z",
		},
		{
			name:     "timezone-less fixture without location",
			fixture:  "../../testdata/logs/synthetic/timestamp-timezone-less.txt",
			wantHold: InstantHoldMissingLogTimezone,
		},
		{
			name:     "dst ambiguous fixture",
			fixture:  "../../testdata/logs/synthetic/timestamp-dst-ambiguous.txt",
			loc:      newYork,
			wantHold: InstantHoldDstAmbiguous,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := fixtureEnvelopeTimestamp(t, tt.fixture)
			got := ResolveCanonicalInstant(raw, tt.loc)
			if got.HoldReason != tt.wantHold {
				t.Fatalf("HoldReason = %q, want %q (canonical=%q)", got.HoldReason, tt.wantHold, got.Canonical)
			}
			if got.Canonical != tt.wantCanon {
				t.Fatalf("Canonical = %q, want %q", got.Canonical, tt.wantCanon)
			}
		})
	}
}

func fixtureEnvelopeTimestamp(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "COMBAT_LOG_VERSION,") || line == "" {
			continue
		}
		split := SplitEnvelope(line)
		if split.Envelope.Raw == "" {
			t.Fatalf("fixture %s has no envelope timestamp", path)
		}
		return split.Envelope.Raw
	}
	t.Fatalf("fixture %s has no timestamped line", path)
	return ""
}
