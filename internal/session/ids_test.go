package session

import (
	"errors"
	"testing"
)

const (
	canonicalRunID = "sha256:8de6d05a987d9230788f5a56957a80c8608335bbc509ca5fac1a1bf17336b30d"
	canonicalGUID  = "Player-0001-00000001"
	runStart       = "2026-02-10T18:30:00.000000000Z"
)

func TestClientRunIDGoldenVectors(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		params   RunIDParams
		expected string
	}{
		{
			// examples/request/minimal-trash-death.json
			name:   "minimal-trash-death",
			source: "minimal-trash-death.json",
			params: RunIDParams{
				ChallengeModeStartInstant: runStart,
				ChallengeMapID:            375,
				KeystoneLevel:             12,
			},
			expected: canonicalRunID,
		},
		{
			// examples/request/boss-context-death.json
			name:   "boss-context-death",
			source: "boss-context-death.json",
			params: RunIDParams{
				ChallengeModeStartInstant: runStart,
				ChallengeMapID:            375,
				KeystoneLevel:             12,
			},
			expected: canonicalRunID,
		},
		{
			// examples/request/repeated-death-after-resurrection.json
			name:   "repeated-death-after-resurrection",
			source: "repeated-death-after-resurrection.json",
			params: RunIDParams{
				ChallengeModeStartInstant: runStart,
				ChallengeMapID:            375,
				KeystoneLevel:             12,
			},
			expected: canonicalRunID,
		},
		{
			// examples/request/mixed-duplicate-needs-review.json
			name:   "mixed-duplicate-needs-review",
			source: "mixed-duplicate-needs-review.json",
			params: RunIDParams{
				ChallengeModeStartInstant: runStart,
				ChallengeMapID:            375,
				KeystoneLevel:             12,
			},
			expected: canonicalRunID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClientRunID(tt.params)
			if err != nil {
				t.Fatalf("ClientRunID(%s): unexpected error: %v", tt.source, err)
			}
			if got != tt.expected {
				t.Fatalf("ClientRunID(%s): got %q want %q", tt.source, got, tt.expected)
			}
		})
	}
}

func TestClientEventIDGoldenVectors(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		params   EventIDParams
		expected string
	}{
		{
			// examples/request/minimal-trash-death.json
			name:   "minimal-trash-death",
			source: "minimal-trash-death.json",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T18:45:12.123456789Z",
				Ordinal:       0,
			},
			expected: "sha256:37655d8725a8b0b2ad06261af3dbd73d17a1441fb17495b1f068f3c2d45cd543",
		},
		{
			// examples/request/boss-context-death.json
			name:   "boss-context-death",
			source: "boss-context-death.json",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T18:52:00.000000000Z",
				Ordinal:       0,
			},
			expected: "sha256:0df4a56b8cc57753750a3215224cb88eae7d54951465c07e6b2b19827e1d8dc0",
		},
		{
			// examples/request/repeated-death-after-resurrection.json (first event)
			name:   "repeated-death-after-resurrection-first",
			source: "repeated-death-after-resurrection.json",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T19:02:33.987654321Z",
				Ordinal:       0,
			},
			expected: "sha256:e7b414d6d92fe83a6ea4e51fb434f821f6c55f59d7ad0c5723703c0e90d1037a",
		},
		{
			// examples/request/repeated-death-after-resurrection.json (second event)
			name:   "repeated-death-after-resurrection-second",
			source: "repeated-death-after-resurrection.json",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T19:15:00.000000000Z",
				Ordinal:       0,
			},
			expected: "sha256:7611877cc6a477bcc000a072a27230f8f2aa333cb3f92a7d3a7f05b2c8418d05",
		},
		{
			// examples/request/mixed-duplicate-needs-review.json (event 1)
			name:   "mixed-duplicate-needs-review-trash",
			source: "mixed-duplicate-needs-review.json",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T18:45:12.123456789Z",
				Ordinal:       0,
			},
			expected: "sha256:37655d8725a8b0b2ad06261af3dbd73d17a1441fb17495b1f068f3c2d45cd543",
		},
		{
			// examples/request/mixed-duplicate-needs-review.json (event 2)
			name:   "mixed-duplicate-needs-review-melee",
			source: "mixed-duplicate-needs-review.json",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T19:02:33.987654321Z",
				Ordinal:       0,
			},
			expected: "sha256:e7b414d6d92fe83a6ea4e51fb434f821f6c55f59d7ad0c5723703c0e90d1037a",
		},
		{
			// examples/request/mixed-duplicate-needs-review.json (event 3)
			name:   "mixed-duplicate-needs-review-boss",
			source: "mixed-duplicate-needs-review.json",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T18:52:00.000000000Z",
				Ordinal:       0,
			},
			expected: "sha256:0df4a56b8cc57753750a3215224cb88eae7d54951465c07e6b2b19827e1d8dc0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClientEventID(tt.params)
			if err != nil {
				t.Fatalf("ClientEventID(%s): unexpected error: %v", tt.source, err)
			}
			if got != tt.expected {
				t.Fatalf("ClientEventID(%s): got %q want %q", tt.source, got, tt.expected)
			}
		})
	}
}

func TestContractDigestFraming(t *testing.T) {
	abC := contractDigest("ab", "c")
	aBC := contractDigest("a", "bc")
	if abC == aBC {
		t.Fatalf("length-prefix framing must distinguish (\"ab\",\"c\") from (\"a\",\"bc\"): both produced %q", abC)
	}
}

func TestAllZeroMismatchDigestsNotProduced(t *testing.T) {
	const allZero = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	runID, err := ClientRunID(RunIDParams{
		ChallengeModeStartInstant: runStart,
		ChallengeMapID:            375,
		KeystoneLevel:             12,
	})
	if err != nil {
		t.Fatalf("ClientRunID: %v", err)
	}
	if runID == allZero {
		t.Fatal("ClientRunID produced all-zero digest for valid run inputs")
	}

	eventID, err := ClientEventID(EventIDParams{
		ClientRunID:   canonicalRunID,
		CharacterGUID: canonicalGUID,
		DeathInstant:  "2026-02-10T18:45:12.123456789Z",
		Ordinal:       0,
	})
	if err != nil {
		t.Fatalf("ClientEventID: %v", err)
	}
	if eventID == allZero {
		t.Fatal("ClientEventID produced all-zero digest for valid event inputs")
	}

	// examples/request/invalid/client-run-id-hash-mismatch.json
	if runID == "sha256:0000000000000000000000000000000000000000000000000000000000000000" {
		t.Fatal("valid run inputs must not match deliberate all-zero clientRunId fixture")
	}

	// examples/request/invalid/client-event-id-hash-mismatch.json
	if eventID == allZero {
		t.Fatal("valid event inputs must not match deliberate all-zero clientEventId fixture")
	}
}

func TestClientRunIDRejectsInvalidInput(t *testing.T) {
	valid := RunIDParams{
		ChallengeModeStartInstant: runStart,
		ChallengeMapID:            375,
		KeystoneLevel:             12,
	}

	tests := []struct {
		name   string
		params RunIDParams
		want   error
	}{
		{
			name: "three fractional digits",
			params: RunIDParams{
				ChallengeModeStartInstant: "2026-02-10T18:30:00.123Z",
				ChallengeMapID:            375,
				KeystoneLevel:             12,
			},
			want: ErrInvalidInstant,
		},
		{
			name: "six fractional digits",
			params: RunIDParams{
				ChallengeModeStartInstant: "2026-02-10T18:30:00.123456Z",
				ChallengeMapID:            375,
				KeystoneLevel:             12,
			},
			want: ErrInvalidInstant,
		},
		{
			name: "offset instead of Z",
			params: RunIDParams{
				ChallengeModeStartInstant: "2026-02-10T18:30:00.000000000+02:00",
				ChallengeMapID:            375,
				KeystoneLevel:             12,
			},
			want: ErrInvalidInstant,
		},
		{
			name: "invalid calendar date",
			params: RunIDParams{
				ChallengeModeStartInstant: "2026-02-30T18:30:00.000000000Z",
				ChallengeMapID:            375,
				KeystoneLevel:             12,
			},
			want: ErrInvalidInstant,
		},
		{
			name: "challenge map id zero",
			params: RunIDParams{
				ChallengeModeStartInstant: runStart,
				ChallengeMapID:            0,
				KeystoneLevel:             12,
			},
			want: ErrInvalidChallengeMap,
		},
		{
			name: "keystone level negative",
			params: RunIDParams{
				ChallengeModeStartInstant: runStart,
				ChallengeMapID:            375,
				KeystoneLevel:             -1,
			},
			want: ErrInvalidKeystoneLevel,
		},
		{
			name: "keystone level above 99",
			params: RunIDParams{
				ChallengeModeStartInstant: runStart,
				ChallengeMapID:            375,
				KeystoneLevel:             100,
			},
			want: ErrInvalidKeystoneLevel,
		},
		{
			name: "non-ascii instant",
			params: RunIDParams{
				ChallengeModeStartInstant: "2026-02-10T18:30:00.00000000\u00e9Z",
				ChallengeMapID:            375,
				KeystoneLevel:             12,
			},
			want: ErrNonASCII,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClientRunID(tt.params)
			if err == nil {
				t.Fatalf("ClientRunID: expected error, got digest %q", got)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("ClientRunID: got error %v, want %v", err, tt.want)
			}
		})
	}

	// Sanity: valid params still work.
	if _, err := ClientRunID(valid); err != nil {
		t.Fatalf("valid ClientRunID params rejected: %v", err)
	}
}

func TestClientEventIDRejectsInvalidInput(t *testing.T) {
	valid := EventIDParams{
		ClientRunID:   canonicalRunID,
		CharacterGUID: canonicalGUID,
		DeathInstant:  "2026-02-10T18:45:12.123456789Z",
		Ordinal:       0,
	}

	tests := []struct {
		name   string
		params EventIDParams
		want   error
	}{
		{
			name: "three fractional digits",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T18:45:12.123Z",
				Ordinal:       0,
			},
			want: ErrInvalidInstant,
		},
		{
			name: "six fractional digits",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T18:45:12.123456Z",
				Ordinal:       0,
			},
			want: ErrInvalidInstant,
		},
		{
			name: "offset instead of Z",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T18:45:12.123456789+02:00",
				Ordinal:       0,
			},
			want: ErrInvalidInstant,
		},
		{
			name: "invalid calendar date",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-30T18:45:12.123456789Z",
				Ordinal:       0,
			},
			want: ErrInvalidInstant,
		},
		{
			name: "negative ordinal",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T18:45:12.123456789Z",
				Ordinal:       -1,
			},
			want: ErrInvalidOrdinal,
		},
		{
			name: "malformed character guid",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: "Creature-0001-00000001",
				DeathInstant:  "2026-02-10T18:45:12.123456789Z",
				Ordinal:       0,
			},
			want: ErrInvalidCharacterGUID,
		},
		{
			name: "malformed client run id",
			params: EventIDParams{
				ClientRunID:   "sha256:NOT_A_VALID_HEX_DIGEST",
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T18:45:12.123456789Z",
				Ordinal:       0,
			},
			want: ErrInvalidClientRunID,
		},
		{
			name: "all-zero client run id from mismatch fixture",
			params: EventIDParams{
				ClientRunID:   "sha256:0000000000000000000000000000000000000000000000000000000000000000",
				CharacterGUID: canonicalGUID,
				DeathInstant:  "2026-02-10T18:45:12.123456789Z",
				Ordinal:       0,
			},
			want: nil,
		},
		{
			name: "non-ascii character guid",
			params: EventIDParams{
				ClientRunID:   canonicalRunID,
				CharacterGUID: "Player-0001-\u00e90000001",
				DeathInstant:  "2026-02-10T18:45:12.123456789Z",
				Ordinal:       0,
			},
			want: ErrNonASCII,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClientEventID(tt.params)
			if tt.want == nil {
				if err != nil {
					t.Fatalf("ClientEventID: expected success for well-formed all-zero clientRunId input, got %v", err)
				}
				if got == "sha256:0000000000000000000000000000000000000000000000000000000000000000" {
					t.Fatal("ClientEventID must not echo all-zero clientEventId for valid hash inputs")
				}
				return
			}
			if err == nil {
				t.Fatalf("ClientEventID: expected error, got digest %q", got)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("ClientEventID: got error %v, want %v", err, tt.want)
			}
		})
	}

	if _, err := ClientEventID(valid); err != nil {
		t.Fatalf("valid ClientEventID params rejected: %v", err)
	}
}
