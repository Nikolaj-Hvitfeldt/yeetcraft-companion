package config

import (
	"encoding/json"
	"errors"
	"os/exec"
	"testing"
	"time"
)

func TestLoadTrackedGUIDs(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    []string
		wantErr error
	}{
		{
			name:    "missing",
			env:     map[string]string{},
			wantErr: ErrNoTrackedCharacters,
		},
		{
			name:    "empty",
			env:     map[string]string{envTrackedGUIDs: ""},
			wantErr: ErrNoTrackedCharacters,
		},
		{
			name:    "comma only",
			env:     map[string]string{envTrackedGUIDs: ","},
			wantErr: ErrNoTrackedCharacters,
		},
		{
			name:    "whitespace only",
			env:     map[string]string{envTrackedGUIDs: "   "},
			wantErr: ErrNoTrackedCharacters,
		},
		{
			name:    "bad guid shape",
			env:     map[string]string{envTrackedGUIDs: "not-a-player-guid"},
			wantErr: ErrInvalidTrackedGUID,
		},
		{
			name: "valid single",
			env:  map[string]string{envTrackedGUIDs: "Player-0001-00000001"},
			want: []string{"Player-0001-00000001"},
		},
		{
			name: "valid multiple",
			env: map[string]string{
				envTrackedGUIDs: "Player-0001-00000001, Player-0002-00000002",
			},
			want: []string{"Player-0001-00000001", "Player-0002-00000002"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(func(key string) (string, bool) {
				value, ok := tt.env[key]
				return value, ok
			})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Load() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if len(cfg.TrackedGUIDs) != len(tt.want) {
				t.Fatalf("TrackedGUIDs = %v, want %v", cfg.TrackedGUIDs, tt.want)
			}
			for i, guid := range tt.want {
				if cfg.TrackedGUIDs[i] != guid {
					t.Fatalf("TrackedGUIDs[%d] = %q, want %q", i, cfg.TrackedGUIDs[i], guid)
				}
			}
		})
	}
}

func TestLoadLogTimezone(t *testing.T) {
	t.Run("unknown timezone", func(t *testing.T) {
		_, err := Load(func(key string) (string, bool) {
			switch key {
			case envTrackedGUIDs:
				return "Player-0001-00000001", true
			case envLogTimezone:
				return "Not/A/Real/Zone", true
			default:
				return "", false
			}
		})
		if !errors.Is(err, ErrInvalidLogTimezone) {
			t.Fatalf("Load() error = %v, want %v", err, ErrInvalidLogTimezone)
		}
	})

	t.Run("valid timezone", func(t *testing.T) {
		cfg, err := Load(func(key string) (string, bool) {
			switch key {
			case envTrackedGUIDs:
				return "Player-0001-00000001", true
			case envLogTimezone:
				return "America/New_York", true
			default:
				return "", false
			}
		})
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if cfg.LogTimezone == nil {
			t.Fatal("LogTimezone is nil")
		}
		if cfg.LogTimezone.String() != "America/New_York" {
			t.Fatalf("LogTimezone = %q", cfg.LogTimezone)
		}
	})

	t.Run("unset timezone", func(t *testing.T) {
		cfg, err := Load(func(key string) (string, bool) {
			if key == envTrackedGUIDs {
				return "Player-0001-00000001", true
			}
			return "", false
		})
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if cfg.LogTimezone != nil {
			t.Fatalf("LogTimezone = %v, want nil", cfg.LogTimezone)
		}
	})
}

func TestLoadInstallationID(t *testing.T) {
	cfg, err := Load(func(key string) (string, bool) {
		switch key {
		case envTrackedGUIDs:
			return "Player-0001-00000001", true
		case envInstallationID:
			return " install-abc ", true
		default:
			return "", false
		}
	})
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.InstallationID != "install-abc" {
		t.Fatalf("InstallationID = %q", cfg.InstallationID)
	}
}

func TestLoadLocationUsesEmbeddedTZData(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Copenhagen")
	if err != nil {
		t.Fatalf("LoadLocation() error = %v", err)
	}
	if loc == nil {
		t.Fatal("LoadLocation returned nil")
	}
}

func TestParserDoesNotImportConfigOrDetection(t *testing.T) {
	cmd := exec.Command("go", "list", "-json", "github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	var info struct {
		ImportPath string
		Imports    []string
	}
	if err := json.Unmarshal(out, &info); err != nil {
		t.Fatalf("decode go list output: %v", err)
	}
	for _, imp := range info.Imports {
		switch imp {
		case "github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/config",
			"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/detection":
			t.Fatalf("internal/parser must not import %s", imp)
		}
	}
}
