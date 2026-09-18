package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/config"
	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/detection"
	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/logwatcher"
	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/session"
	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/storage"
)

const (
	envCompanionDBPath  = "COMPANION_DB_PATH"
	defaultPollInterval = 250 * time.Millisecond
)

type captureOptions struct {
	LogFile           string
	DBPath            string
	Once              bool
	PollInterval      time.Duration
	InactivityTimeout time.Duration
	Lookup            func(string) (string, bool)
}

type captureService struct {
	cfg              config.Config
	db               *storage.DB
	tracker          *detection.Tracker
	session          *session.Manager
	watcher          *logwatcher.Watcher
	persistedDeaths  int
	lastPersistedRun string
}

func runCapture(ctx context.Context, opts captureOptions, stderr io.Writer) int {
	cfg, err := config.Load(opts.Lookup)
	if err != nil {
		writeConfigError(stderr, err)
		return exitConfig
	}

	dbPath := opts.DBPath
	if dbPath == "" {
		if value, ok := opts.Lookup(envCompanionDBPath); ok && strings.TrimSpace(value) != "" {
			dbPath = strings.TrimSpace(value)
		}
	}
	if dbPath == "" {
		dbPath = filepath.Join("data", "companion.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		fmt.Fprintf(stderr, "create database directory: %v\n", err)
		return exitFailure
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open database: %v\n", err)
		return exitFailure
	}
	defer db.Close()

	if err := persistSettings(ctx, db, cfg); err != nil {
		fmt.Fprintf(stderr, "persist settings: %v\n", err)
		return exitFailure
	}

	tracker, err := detection.NewTracker(cfg.TrackedGUIDs, cfg.LogTimezone)
	if err != nil {
		fmt.Fprintf(stderr, "tracker: %v\n", err)
		return exitFailure
	}

	sessionMgr := session.NewManager(opts.InactivityTimeout)
	if err := restoreActiveRun(ctx, db, sessionMgr, tracker); err != nil {
		fmt.Fprintf(stderr, "restore active run: %v\n", err)
		return exitFailure
	}

	identity, _, err := logwatcher.ResolveIdentity(opts.LogFile)
	if err != nil {
		fmt.Fprintf(stderr, "resolve combat log identity: %v\n", err)
		return exitFailure
	}

	committed, err := loadCommittedOffset(ctx, db, opts.LogFile, identity)
	if err != nil {
		fmt.Fprintf(stderr, "load file offset: %v\n", err)
		return exitFailure
	}

	service := &captureService{
		cfg:     cfg,
		db:      db,
		tracker: tracker,
		session: sessionMgr,
		watcher: logwatcher.New(opts.LogFile, committed, identity),
	}

	if opts.Once {
		if err := service.pollOnce(ctx); err != nil {
			fmt.Fprintf(stderr, "capture poll: %v\n", err)
			return exitFailure
		}
		return exitOK
	}

	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if closed := service.session.Close(); closed != nil {
				if err := service.persistRunClosure(ctx, closed, storage.RunStatusAbandoned); err != nil {
					fmt.Fprintf(stderr, "close run: %v\n", err)
					return exitFailure
				}
			}
			return exitOK
		case <-ticker.C:
			if err := service.pollOnce(ctx); err != nil {
				fmt.Fprintf(stderr, "capture poll: %v\n", err)
				return exitFailure
			}
			if abandoned := service.session.CheckInactivity(); abandoned != nil {
				if err := service.persistRunClosure(ctx, abandoned, storage.RunStatusAbandoned); err != nil {
					fmt.Fprintf(stderr, "abandon run: %v\n", err)
					return exitFailure
				}
			}
			if completed := service.session.TakeCompleted(); completed != nil {
				if err := service.persistRunClosure(ctx, completed, storage.RunStatusCompleted); err != nil {
					fmt.Fprintf(stderr, "complete run: %v\n", err)
					return exitFailure
				}
			}
		}
	}
}

func (s *captureService) pollOnce(ctx context.Context) error {
	result, err := s.watcher.Poll(ctx, func(event parser.Event) error {
		s.session.Observe(event, s.cfg.LogTimezone)
		return s.tracker.Observe(event)
	})
	if err != nil {
		return err
	}

	if err := s.persistPoll(ctx, result); err != nil {
		return err
	}
	s.watcher.AckCommitted(result.Committed)
	return nil
}

func (s *captureService) persistPoll(ctx context.Context, result logwatcher.PollResult) error {
	current := s.session.Current()
	deaths := s.tracker.Deaths()
	newDeaths := deaths[s.persistedDeaths:]
	s.persistedDeaths = len(deaths)

	if result.Summary.BytesConsumed == 0 && result.Summary.LinesComplete == 0 && len(newDeaths) == 0 {
		return nil
	}

	parserStateJSON, err := parser.EncodeParserState(result.Committed.ParserState)
	if err != nil {
		return err
	}

	fileState := storage.FileState{
		Path:            s.watcher.Path,
		FileIdentity:    s.watcher.Identity().StorageKey(),
		Generation:      result.Committed.Generation,
		ByteOffset:      result.Committed.ByteOffset,
		PartialLine:     result.Committed.PartialLine,
		ParserStateJSON: parserStateJSON,
	}

	events := make([]storage.EventInput, 0, len(newDeaths))
	for _, death := range newDeaths {
		eventInput, err := deathToEventInput(death)
		if err != nil {
			return err
		}
		events = append(events, eventInput)
	}

	runInput := storage.RunInput{}
	if current != nil && current.ClientRunID != "" {
		runInput = storage.RunInput{
			ClientRunID:               current.ClientRunID,
			ChallengeModeStartInstant: current.ChallengeModeStartInstant,
			ChallengeMapID:            current.ChallengeMapID,
			KeystoneLevel:             current.KeystoneLevel,
			Status:                    storage.RunStatusActive,
			StartedAt:                 current.StartedAt,
		}
		if s.lastPersistedRun != current.ClientRunID {
			s.lastPersistedRun = current.ClientRunID
		}
	} else if len(events) > 0 {
		return fmt.Errorf("persist death events without active run")
	}

	return s.db.Commit(ctx, storage.CommitInput{
		Run:    runInput,
		Events: events,
		File:   fileState,
	})
}

func (s *captureService) persistRunClosure(ctx context.Context, snapshot *session.RunSnapshot, status string) error {
	if snapshot == nil || snapshot.ClientRunID == "" {
		return nil
	}
	endedAt := s.sessionNow(snapshot)
	return s.db.UpdateRunStatus(ctx, snapshot.ClientRunID, status, endedAt)
}

func (s *captureService) sessionNow(snapshot *session.RunSnapshot) time.Time {
	if snapshot.EndedAt != nil {
		return *snapshot.EndedAt
	}
	return time.Now().UTC()
}

func deathToEventInput(death detection.DeathCandidate) (storage.EventInput, error) {
	holdReason := string(death.HoldReason)
	if death.DeathInstantHold != "" {
		if holdReason != "" {
			holdReason += ";"
		}
		holdReason += string(death.DeathInstantHold)
	}

	clientEventID := fmt.Sprintf("local:held:%d:%s", death.LineNumber, death.VictimGUID)
	reviewStatus := storage.ReviewStatusHeld
	if death.CanComputeIDs() {
		id, err := death.ClientEventID()
		if err != nil {
			return storage.EventInput{}, err
		}
		clientEventID = id
		reviewStatus = storage.ReviewStatusPending
	}

	causes := make([]storage.CauseInput, 0, len(death.Causes))
	for _, candidate := range death.Causes {
		causes = append(causes, storage.CauseInput{
			Rank:              candidate.Rank,
			SourceType:        string(candidate.Cause.SourceType),
			SpellID:           candidate.Cause.SpellID,
			CreatureID:        candidate.Cause.CreatureID,
			HasCreatureID:     candidate.Cause.HasCreatureID,
			EnvironmentalType: candidate.Cause.EnvironmentalType,
			Amount:            candidate.Cause.Amount,
			Overkill:          candidate.Cause.Overkill,
			PlayerOrigin:      candidate.Cause.PlayerOrigin,
			Confidence:        string(candidate.Confidence),
		})
	}

	return storage.EventInput{
		ClientEventID: clientEventID,
		CharacterGUID: death.VictimGUID,
		DeathInstant:  death.DeathInstant,
		Ordinal:       death.Ordinal,
		HoldReason:    holdReason,
		ReviewStatus:  reviewStatus,
		Confidence:    string(confidenceForDeath(death)),
		Causes:        causes,
	}, nil
}

func confidenceForDeath(death detection.DeathCandidate) detection.Confidence {
	if len(death.Causes) == 0 {
		return detection.ConfidenceLow
	}
	return death.Causes[0].Confidence
}

func persistSettings(ctx context.Context, db *storage.DB, cfg config.Config) error {
	if cfg.InstallationID != "" {
		if err := db.SetSetting(ctx, storage.SettingInstallationID, cfg.InstallationID); err != nil {
			return err
		}
	}
	if cfg.LogTimezone != nil {
		if err := db.SetSetting(ctx, storage.SettingWoWLogTimezone, cfg.LogTimezone.String()); err != nil {
			return err
		}
	}
	return nil
}

func restoreActiveRun(ctx context.Context, db *storage.DB, sessionMgr *session.Manager, tracker *detection.Tracker) error {
	active, err := db.GetActiveRun(ctx)
	if err != nil {
		return err
	}
	if active == nil {
		return nil
	}
	sessionMgr.RestoreActiveRun(
		active.ClientRunID,
		active.ChallengeModeStartInstant,
		active.ChallengeMapID,
		active.KeystoneLevel,
		active.StartedAt,
	)
	tracker.RestoreRun(detection.RunContext{
		Active:        true,
		MapID:         active.ChallengeMapID,
		KeystoneLevel: active.KeystoneLevel,
		StartInstant:  active.ChallengeModeStartInstant,
	})
	return nil
}

func loadCommittedOffset(ctx context.Context, db *storage.DB, path string, identity logwatcher.Identity) (logwatcher.CommittedOffset, error) {
	state, err := db.GetFileState(ctx, path, identity.StorageKey())
	if err != nil {
		return logwatcher.CommittedOffset{}, err
	}
	if state == nil {
		return logwatcher.ResetOffset(0), nil
	}

	parserState, err := parser.DecodeParserState(state.ParserStateJSON)
	if err != nil {
		return logwatcher.CommittedOffset{}, err
	}
	return logwatcher.CommittedOffset{
		Generation:  state.Generation,
		ByteOffset:  state.ByteOffset,
		PartialLine: state.PartialLine,
		ParserState: parserState,
	}, nil
}

func writeConfigError(w io.Writer, err error) {
	fmt.Fprintf(w, "configuration error: %v\n", err)
	switch {
	case errors.Is(err, config.ErrNoTrackedCharacters):
		fmt.Fprintln(w, "Set YEETCRAFT_TRACKED_GUIDS to a comma-separated list of Player- prefixed character GUIDs.")
	case errors.Is(err, config.ErrInvalidTrackedGUID):
		fmt.Fprintln(w, "Every tracked GUID must match the Player-<server>-<id> shape.")
	case errors.Is(err, config.ErrInvalidLogTimezone):
		fmt.Fprintln(w, "Set WOW_LOG_TIMEZONE to a valid IANA timezone name.")
	}
}

func parseDurationFlag(raw string, fallback time.Duration) (time.Duration, error) {
	if raw == "" {
		return fallback, nil
	}
	return time.ParseDuration(raw)
}

func parsePollInterval(raw string) (time.Duration, error) {
	if raw == "" {
		return defaultPollInterval, nil
	}
	if strings.HasSuffix(raw, "ms") || strings.HasSuffix(raw, "s") || strings.HasSuffix(raw, "m") {
		return time.ParseDuration(raw)
	}
	ms, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid poll interval %q", raw)
	}
	return time.Duration(ms) * time.Millisecond, nil
}
