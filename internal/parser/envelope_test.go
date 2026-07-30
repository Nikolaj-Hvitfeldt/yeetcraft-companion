package parser

import "testing"

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
