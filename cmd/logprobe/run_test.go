package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
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
	} {
		t.Run(fixture, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exit := run([]string{"--file", fixturePath(fixture)}, &stdout, &stderr)
			if exit != exitOK && exit != exitUnsupported {
				t.Fatalf("exit = %d; stderr=%q", exit, stderr.String())
			}
			for _, sensitive := range []string{
				"Synthetic Encounter",
				"Synthetic Source",
				"Synthetic Target",
				"Quarantined Source",
				"Creature-Synthetic",
				"Player-Synthetic",
				"SYNTHETIC_UNKNOWN_EVENT",
				"COMBAT_LOG_VERSION",
			} {
				if strings.Contains(stdout.String(), sensitive) {
					t.Fatalf("stdout exposed %q: %q", sensitive, stdout.String())
				}
			}
		})
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
