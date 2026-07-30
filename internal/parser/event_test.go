package parser

import "testing"

func supportedState() *ParserState {
	return &ParserState{Format: FormatStateSupportedV22, FormatVersion: SupportedFormatVersion}
}

func TestEventRecognitionIsExplicit(t *testing.T) {
	known := `SPELL_DAMAGE,source-guid,"Source",0xa48,0x0,dest-guid,"Destination",0x512,0x0,trailing`
	event := ParseLine(1, known, supportedState())
	if event.Kind != KindCommonHeader || event.Common == nil || event.Structural != nil {
		t.Fatalf("known event = %#v", event)
	}

	unknown := `SYNTHETIC_UNKNOWN_EVENT,source-guid,"Source",0xa48,0x0,dest-guid,"Destination",0x512,0x0,trailing`
	event = ParseLine(2, unknown, supportedState())
	if event.Kind != KindUnknownEvent {
		t.Fatalf("unknown kind = %v, want KindUnknownEvent", event.Kind)
	}
	if event.Common != nil || event.Structural == nil {
		t.Fatalf("unknown common=%#v structural=%#v", event.Common, event.Structural)
	}
	if len(event.Fields) != 10 || event.Fields[9] != "trailing" {
		t.Fatalf("unknown fields not preserved: %#v", event.Fields)
	}
}

func TestCommonHeaderExtraction(t *testing.T) {
	line := `UNIT_DIED,0000000000000000,nil,0x80000000,0x80000000,Player-Synthetic,"Synthetic Target",0x512,0x0,tail`
	event := ParseLine(1, line, supportedState())
	if event.Kind != KindCommonHeader || event.Common == nil {
		t.Fatalf("event = %#v", event)
	}
	if !event.Common.SourceNameIsNil || event.Common.SourceName != "" {
		t.Fatalf("nil source name = %#v", event.Common)
	}
	if event.Common.SourceFlags == nil || *event.Common.SourceFlags != 0x80000000 {
		t.Fatalf("source flags = %#v", event.Common.SourceFlags)
	}
	if event.Common.DestFlags == nil || *event.Common.DestFlags != 0x512 {
		t.Fatalf("dest flags = %#v", event.Common.DestFlags)
	}
}

func TestInvalidCommonHeaderFlagsAreWarnings(t *testing.T) {
	line := `SPELL_DAMAGE,source,"Source",invalid,0x0,dest,"Destination",0x512,bad`
	event := ParseLine(1, line, supportedState())
	if event.Kind != KindCommonHeader {
		t.Fatalf("kind = %v", event.Kind)
	}
	if event.Common.SourceFlags != nil || event.Common.DestRaidFlags != nil || len(event.Warnings) != 2 {
		t.Fatalf("common=%#v warnings=%#v", event.Common, event.Warnings)
	}
}

func TestCommonHeaderFlagsUseStrictHexTokens(t *testing.T) {
	line := `UNIT_DIED,source,"Source",0X1,0xffffffff,dest,"Destination",10,010`
	event := ParseLine(1, line, supportedState())
	if event.Kind != KindCommonHeader || event.Common == nil {
		t.Fatalf("event = %#v", event)
	}
	if event.Common.SourceFlags == nil || *event.Common.SourceFlags != 1 {
		t.Fatalf("source flags = %#v", event.Common.SourceFlags)
	}
	if event.Common.SourceRaidFlags == nil || *event.Common.SourceRaidFlags != 0xffffffff {
		t.Fatalf("source raid flags = %#v", event.Common.SourceRaidFlags)
	}
	if event.Common.DestFlags != nil || event.Common.DestRaidFlags != nil || len(event.Warnings) != 2 {
		t.Fatalf("common=%#v warnings=%#v", event.Common, event.Warnings)
	}
}

func TestKnownCommonHeaderTooShortIsMalformed(t *testing.T) {
	event := ParseLine(1, `SPELL_DAMAGE,too,few`, supportedState())
	if event.Kind != KindMalformed || event.Err == nil {
		t.Fatalf("event = %#v", event)
	}
}

func TestMetadataHasNoCommonHeader(t *testing.T) {
	event := ParseLine(1, `ENCOUNTER_START,990001,"Synthetic Encounter",8,5,9999`, supportedState())
	if event.Kind != KindMetadata || event.Common != nil || event.Structural != nil {
		t.Fatalf("event = %#v", event)
	}
}

func TestQuarantinedRowsRemainGeneric(t *testing.T) {
	state := &ParserState{Format: FormatStateQuarantinedUnsupported}
	event := ParseLine(1, `SPELL_DAMAGE,source,"Source",0xa48,0x0,dest,"Destination",0x512,0x0`, state)
	if event.Kind != KindGeneric || event.Common != nil || event.Structural != nil {
		t.Fatalf("event = %#v", event)
	}
}
