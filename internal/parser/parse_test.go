package parser

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseMalformedCSVIsControlled(t *testing.T) {
	event := ParseLine(7, `SPELL_DAMAGE,"unterminated`, supportedState())
	if event.Kind != KindMalformed || !errors.Is(event.Err, ErrMalformedCSV) {
		t.Fatalf("event = %#v", event)
	}
	var lineErr *LineError
	if !errors.As(event.Err, &lineErr) || lineErr.LineNumber != 7 {
		t.Fatalf("line error = %#v", lineErr)
	}
}

func TestParseLineDoesNotPanicOnMalformedInput(t *testing.T) {
	inputs := []string{"", ",", `"`, "\x00\x01", "EVENT", "EVENT,a,b,c,d,e,f,g,h"}
	for i, input := range inputs {
		_ = ParseLine(i+1, input, supportedState())
	}
}

func TestFocusedSyntheticFixtures(t *testing.T) {
	tests := []struct {
		name             string
		wantLastKind     Kind
		wantFormat       FormatState
		wantVersionCount int
	}{
		{"csv-quoted-fields.txt", KindUnknownEvent, FormatStateSupportedV22, 1},
		{"common-header-invalid-flags.txt", KindCommonHeader, FormatStateSupportedV22, 1},
		{"common-header-too-few-fields.txt", KindMalformed, FormatStateSupportedV22, 1},
		{"version-malformed-kv.txt", KindMalformed, FormatStateQuarantinedUnsupported, 1},
		{"version-repeated-boundary.txt", KindVersionHeader, FormatStateSupportedV22, 2},
		{"version-v22-then-unsupported.txt", KindGeneric, FormatStateQuarantinedUnsupported, 3},
		{"version-v22-then-malformed.txt", KindGeneric, FormatStateQuarantinedUnsupported, 3},
		{"version-project-id-2.txt", KindVersionHeader, FormatStateQuarantinedUnsupported, 1},
		{"version-project-id-non-integer.txt", KindMalformed, FormatStateQuarantinedUnsupported, 1},
		{"timestamp-signed-offset.txt", KindUnknownEvent, FormatStateSupportedV22, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "logs", "synthetic", tt.name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			state := &ParserState{}
			var last Event
			_, err = ScanReader(bytes.NewReader(data), DefaultMaxLineSize, state, func(event Event) error {
				last = event
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if last.Kind != tt.wantLastKind || state.Format != tt.wantFormat || state.VersionHeaders != tt.wantVersionCount {
				t.Fatalf("last kind=%v state=%#v", last.Kind, state)
			}
		})
	}
}
