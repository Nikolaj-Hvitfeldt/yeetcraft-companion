package parser

import (
	"errors"
	"testing"
)

func TestParseVersionHeader(t *testing.T) {
	tests := []struct {
		name         string
		fields       []string
		wantFormat   int
		wantErr      bool
		wantErrIs    error
		wantAdvanced bool
	}{
		{
			"supported V22",
			[]string{"COMBAT_LOG_VERSION", "22", "ADVANCED_LOG_ENABLED", "1", "BUILD_VERSION", "12.0.0", "PROJECT_ID", "1"},
			22,
			false,
			nil,
			true,
		},
		{
			"supported V22 advanced disabled",
			[]string{"COMBAT_LOG_VERSION", "22", "ADVANCED_LOG_ENABLED", "0", "BUILD_VERSION", "12.0.0", "PROJECT_ID", "1"},
			22,
			false,
			nil,
			false,
		},
		{
			"unsupported V99",
			[]string{"COMBAT_LOG_VERSION", "99", "ADVANCED_LOG_ENABLED", "1", "BUILD_VERSION", "99.0.0", "PROJECT_ID", "1"},
			99,
			true,
			ErrUnsupportedFormatVersion,
			true,
		},
		{
			"unsupported project",
			[]string{"COMBAT_LOG_VERSION", "22", "ADVANCED_LOG_ENABLED", "1", "BUILD_VERSION", "12.0.0", "PROJECT_ID", "2"},
			22,
			true,
			ErrUnsupportedProject,
			true,
		},
		{
			"malformed key value",
			[]string{"COMBAT_LOG_VERSION", "22", "WRONG_KEY", "1"},
			0,
			true,
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersionHeader(tt.fields)
			if !tt.wantErr && err != nil {
				t.Fatalf("ParseVersionHeader() error = %v", err)
			}
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("error = %v, want %v", err, tt.wantErrIs)
			}
			if got.FormatVersion != tt.wantFormat {
				t.Fatalf("format = %d, want %d", got.FormatVersion, tt.wantFormat)
			}
			if !tt.wantErr && got.AdvancedLogEnabled != tt.wantAdvanced {
				t.Fatalf("advanced = %v, want %v", got.AdvancedLogEnabled, tt.wantAdvanced)
			}
		})
	}
}

func TestVersionStateQuarantineDoesNotRecover(t *testing.T) {
	state := &ParserState{}
	v22 := "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1"
	v99 := "COMBAT_LOG_VERSION,99,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,99.0.0,PROJECT_ID,1"

	ParseLine(1, v22, state)
	ParseLine(2, v22, state)
	if state.Format != FormatStateSupportedV22 || state.SupportedHeaders != 2 {
		t.Fatalf("state after V22 repeat = %#v", state)
	}
	event := ParseLine(3, v99, state)
	if event.Kind != KindVersionHeader || !errors.Is(event.Err, ErrUnsupportedFormatVersion) {
		t.Fatalf("unsupported event = %#v", event)
	}
	ParseLine(4, v22, state)
	if state.Format != FormatStateQuarantinedUnsupported {
		t.Fatalf("state recovered unexpectedly: %#v", state)
	}
	if state.VersionHeaders != 4 || state.SupportedHeaders != 3 || state.UnsupportedHeaders != 1 {
		t.Fatalf("version counts = %#v", state)
	}
}

func TestMalformedVersionBoundaryQuarantinesAndNeverRecovers(t *testing.T) {
	v22 := "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1"
	malformed := "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,invalid,BUILD_VERSION,12.0.0,PROJECT_ID,1"
	normal := `SPELL_DAMAGE,source,"Source",0xa48,0x0,dest,"Destination",0x512,0x0`

	tests := []struct {
		name   string
		prefix []string
	}{
		{"after V22", []string{v22}},
		{"as first record", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &ParserState{}
			line := 1
			for _, record := range tt.prefix {
				ParseLine(line, record, state)
				line++
			}
			boundary := ParseLine(line, malformed, state)
			if boundary.Kind != KindMalformed || boundary.Malformed != MalformedVersionHeader {
				t.Fatalf("boundary = %#v", boundary)
			}
			if state.Format != FormatStateQuarantinedUnsupported || state.MalformedHeaders != 1 {
				t.Fatalf("state after malformed boundary = %#v", state)
			}
			if event := ParseLine(line+1, normal, state); event.Kind != KindGeneric {
				t.Fatalf("post-boundary kind = %v, want generic", event.Kind)
			}
			ParseLine(line+2, v22, state)
			if state.Format != FormatStateQuarantinedUnsupported {
				t.Fatalf("later V22 recovered state: %#v", state)
			}
		})
	}
}

func TestMalformedVersionFormsQuarantine(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"incorrect field count", "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1"},
		{"incorrect key", "COMBAT_LOG_VERSION,22,WRONG_KEY,1,BUILD_VERSION,12.0.0,PROJECT_ID,1"},
		{"invalid format version", "COMBAT_LOG_VERSION,not-an-int,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1"},
		{"invalid advanced marker", "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,2,BUILD_VERSION,12.0.0,PROJECT_ID,1"},
		{"invalid project ID", "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,not-an-int"},
		{"malformed CSV quoting", `COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,"unterminated`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &ParserState{}
			event := ParseLine(1, tt.header, state)
			if event.Kind != KindMalformed || event.Malformed != MalformedVersionHeader {
				t.Fatalf("event = %#v", event)
			}
			if state.Format != FormatStateQuarantinedUnsupported || state.MalformedHeaders != 1 {
				t.Fatalf("state = %#v", state)
			}
		})
	}
}

func TestUnsupportedProjectQuarantines(t *testing.T) {
	state := &ParserState{}
	event := ParseLine(1, "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,2", state)
	if event.Kind != KindVersionHeader || !errors.Is(event.Err, ErrUnsupportedProject) {
		t.Fatalf("event = %#v", event)
	}
	if state.Format != FormatStateQuarantinedUnsupported || state.UnsupportedProjects != 1 {
		t.Fatalf("state = %#v", state)
	}
}
