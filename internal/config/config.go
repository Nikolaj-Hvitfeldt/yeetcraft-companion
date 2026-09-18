package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "time/tzdata"
)

const (
	envTrackedGUIDs   = "YEETCRAFT_TRACKED_GUIDS"
	envLogTimezone    = "WOW_LOG_TIMEZONE"
	envInstallationID = "COMPANION_INSTALLATION_ID"
)

// ErrNoTrackedCharacters is returned when tracked-character configuration is missing or empty.
var ErrNoTrackedCharacters = errors.New("no tracked characters configured")

// ErrInvalidTrackedGUID is returned when a tracked GUID does not match the Player- prefix shape.
var ErrInvalidTrackedGUID = errors.New("invalid tracked player guid")

// ErrInvalidLogTimezone is returned when WOW_LOG_TIMEZONE is set but not a known IANA zone.
var ErrInvalidLogTimezone = errors.New("invalid log timezone")

var playerGUIDPattern = regexp.MustCompile(`^Player-[0-9A-Za-z]+-[0-9A-Za-z]+$`)

// Config holds companion application settings loaded from the environment.
// InstallationID is diagnostics-only and is never used as a hash input or credential.
type Config struct {
	TrackedGUIDs   []string
	LogTimezone    *time.Location
	InstallationID string
}

// Load reads configuration from lookup, which should return environment variable values.
// Tracked GUIDs are required; invalid GUID shape is an error, not a silent skip.
// When WOW_LOG_TIMEZONE is set, it must name a known IANA timezone.
func Load(lookup func(string) (string, bool)) (Config, error) {
	rawGUIDs, ok := lookup(envTrackedGUIDs)
	if !ok || strings.TrimSpace(rawGUIDs) == "" {
		return Config{}, ErrNoTrackedCharacters
	}

	trackedGUIDs, err := parseTrackedGUIDs(rawGUIDs)
	if err != nil {
		return Config{}, err
	}

	var logTimezone *time.Location
	if tzName, ok := lookup(envLogTimezone); ok {
		tzName = strings.TrimSpace(tzName)
		if tzName != "" {
			loc, err := time.LoadLocation(tzName)
			if err != nil {
				return Config{}, fmt.Errorf("%w: %s", ErrInvalidLogTimezone, tzName)
			}
			logTimezone = loc
		}
	}

	installationID := ""
	if id, ok := lookup(envInstallationID); ok {
		installationID = strings.TrimSpace(id)
	}

	return Config{
		TrackedGUIDs:   trackedGUIDs,
		LogTimezone:    logTimezone,
		InstallationID: installationID,
	}, nil
}

func parseTrackedGUIDs(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	tracked := make([]string, 0, len(parts))
	for _, part := range parts {
		guid := strings.TrimSpace(part)
		if guid == "" {
			continue
		}
		if !playerGUIDPattern.MatchString(guid) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTrackedGUID, guid)
		}
		tracked = append(tracked, guid)
	}
	if len(tracked) == 0 {
		return nil, ErrNoTrackedCharacters
	}
	return tracked, nil
}
