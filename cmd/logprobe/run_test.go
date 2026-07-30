package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

func fixturePath(name string) string {
	return filepath.Join("..", "..", "testdata", "logs", "synthetic", name)
}

func TestRunFixtureExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantExit int
		contains string
	}{
		{"valid", "parser-smoke-valid.txt", exitOK, "format_state: supported"},
		{"unknown", "unknown-event.txt", exitOK, "unknown_event_types: 1"},
		{"malformed", "malformed-csv.txt", exitOK, "malformed_csv: 1"},
		{"truncated", "truncated-line.txt", exitIncomplete, "incomplete_trailing: 1"},
		{"unsupported", "unsupported-version.txt", exitUnsupported, "format_state: quarantined"},
		{"quarantine", "version-v22-then-unsupported.txt", exitUnsupported, "events_generic: 2"},
		{"malformed version quarantine", "version-v22-then-malformed.txt", exitUnsupported, "malformed_version_headers: 1"},
		{"unsupported project", "version-project-id-2.txt", exitUnsupported, "version_headers_unsupported_project: 1"},
		{"non-integer project", "version-project-id-non-integer.txt", exitUnsupported, "malformed_version_headers: 1"},
		{"signed timestamp offset", "timestamp-signed-offset.txt", exitOK, "unknown_event_types: 1"},
		{"malformed common header", "common-header-too-few-fields.txt", exitOK, "malformed_common_headers: 1"},
		{"typed damage", "typed-damage-v22.txt", exitOK, "typed_payloads_parsed: 5"},
		{"typed metadata", "typed-metadata-v22.txt", exitOK, "typed_challenge_mode_end_parsed: 2"},
		{"typed invalid and diagnostics", "typed-payload-invalid-v22.txt", exitOK, "validation_diagnostics: 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exit := run([]string{"--file", fixturePath(tt.fixture)}, &stdout, &stderr)
			if exit != tt.wantExit {
				t.Fatalf("exit = %d, want %d; stdout=%q stderr=%q", exit, tt.wantExit, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), tt.contains) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.contains)
			}
		})
	}
}

func TestRunOutputIsPrivacySafe(t *testing.T) {
	for _, fixture := range []string{
		"parser-smoke-valid.txt",
		"version-v22-then-malformed.txt",
		"timestamp-signed-offset.txt",
		"typed-damage-v22.txt",
		"typed-metadata-v22.txt",
		"typed-payload-invalid-v22.txt",
	} {
		t.Run(fixture, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exit := run([]string{"--file", fixturePath(fixture)}, &stdout, &stderr)
			if exit != exitOK && exit != exitUnsupported {
				t.Fatalf("exit = %d; stderr=%q", exit, stderr.String())
			}
			output := stdout.String() + stderr.String()
			for _, sensitive := range []string{
				"Synthetic Encounter",
				"Synthetic Source",
				"Synthetic Target",
				"Quarantined Source",
				"Creature-Synthetic",
				"Player-Synthetic",
				"SYNTHETIC_UNKNOWN_EVENT",
				"COMBAT_LOG_VERSION",
				"FUTURE_HINT",
				"FutureHazard",
				"Synthetic Spell",
				"Synthetic Encounter",
			} {
				if strings.Contains(output, sensitive) {
					t.Fatalf("output exposed %q: %q", sensitive, output)
				}
			}
		})
	}
}

func TestRunSeparatesTypedErrorsAndDiagnostics(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := run([]string{"--file", fixturePath("typed-payload-invalid-v22.txt")}, &stdout, &stderr)
	if exit != exitOK {
		t.Fatalf("exit = %d; stderr=%q", exit, stderr.String())
	}
	for _, expected := range []string{
		"typed_payloads_parsed: 2",
		"typed_payloads_invalid: 1",
		"typed_error_integer: 1",
		"validation_diagnostics: 3",
		"validation_advanced_info_guid_mismatch: 1",
		"validation_ability_hint_unknown: 1",
		"validation_environmental_type_unknown: 1",
	} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), expected)
		}
	}
}

func TestRunMalformedCountersAreSpecific(t *testing.T) {
	tests := []struct {
		fixture string
		want    string
	}{
		{"malformed-csv.txt", "malformed_csv: 1"},
		{"version-malformed-kv.txt", "malformed_version_headers: 1"},
		{"common-header-too-few-fields.txt", "malformed_common_headers: 1"},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			run([]string{"--file", fixturePath(tt.fixture)}, &stdout, &stderr)
			if !strings.Contains(stdout.String(), tt.want) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.want)
			}
			for _, other := range []string{"malformed_csv: 1", "malformed_version_headers: 1", "malformed_common_headers: 1"} {
				if other != tt.want && strings.Contains(stdout.String(), other) {
					t.Fatalf("stdout incorrectly includes %q: %q", other, stdout.String())
				}
			}
		})
	}
}

func TestRunHelpAndUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exit := run([]string{"--help"}, &stdout, &stderr); exit != exitOK {
		t.Fatalf("help exit = %d", exit)
	}
	if !strings.Contains(stderr.String(), "Usage: logprobe --file <path>") {
		t.Fatalf("help = %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if exit := run(nil, &stdout, &stderr); exit != exitFailure {
		t.Fatalf("missing file exit = %d", exit)
	}
}

func TestRunOpenFailureDoesNotExposePath(t *testing.T) {
	privatePath := `C:\Users\PrivatePerson\WoWCombatLog-PrivateCharacter.txt`
	var stdout, stderr bytes.Buffer
	if exit := run([]string{"--file", privatePath}, &stdout, &stderr); exit != exitFailure {
		t.Fatalf("exit = %d", exit)
	}
	output := stdout.String() + stderr.String()
	for _, sensitive := range []string{privatePath, "PrivatePerson", "PrivateCharacter"} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("output exposed %q: %q", sensitive, output)
		}
	}
	if !strings.Contains(stderr.String(), "unable to open requested file") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestWriteScanErrorDoesNotExposeWrappedPath(t *testing.T) {
	privatePath := `C:\Users\PrivatePerson\WoWCombatLog-PrivateCharacter.txt`
	wrapped := fmt.Errorf("read combat log: %w", &os.PathError{
		Op:   "read",
		Path: privatePath,
		Err:  errors.New("fictional private failure"),
	})
	var stdout, stderr bytes.Buffer
	writeScanError(&stderr, wrapped)

	output := stdout.String() + stderr.String()
	for _, sensitive := range []string{
		privatePath,
		"PrivatePerson",
		"PrivateCharacter",
		"fictional private failure",
		wrapped.Error(),
	} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("output exposed %q: %q", sensitive, output)
		}
	}
	if stderr.String() != "scan combat log: read or parse failure\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestClassifyScanErrorUsesFixedCategories(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"oversized", fmt.Errorf("wrapped: %w", parser.ErrLineTooLong), "scan combat log: oversized line"},
		{"handler", fmt.Errorf("wrapped: %w", parser.ErrEventHandler), "scan combat log: event handler failed"},
		{"generic", errors.New("private raw error"), "scan combat log: read or parse failure"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyScanError(tt.err); got != tt.want {
				t.Fatalf("classifyScanError() = %q, want %q", got, tt.want)
			}
		})
	}
}
