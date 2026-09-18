package session

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

const (
	domainTagRun   = "yeetcraft-run-v1"
	domainTagDeath = "yeetcraft-death-v1"
)

var (
	ErrInvalidInstant       = errors.New("invalid canonical instant")
	ErrInvalidOrdinal       = errors.New("invalid ordinal")
	ErrInvalidChallengeMap  = errors.New("invalid challenge map id")
	ErrInvalidKeystoneLevel = errors.New("invalid keystone level")
	ErrInvalidCharacterGUID = errors.New("invalid character guid")
	ErrInvalidClientRunID   = errors.New("invalid client run id")
	ErrNonASCII             = errors.New("non-ascii field")
)

var (
	canonicalInstantPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{9}Z$`)
	clientRunIDPattern      = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	playerGUIDPattern       = regexp.MustCompile(`^Player-[0-9A-Za-z]+-[0-9A-Za-z]+$`)
)

// RunIDParams holds the hash inputs for clientRunId.
type RunIDParams struct {
	ChallengeModeStartInstant string
	ChallengeMapID            int
	KeystoneLevel             int
}

// EventIDParams holds the hash inputs for clientEventId.
type EventIDParams struct {
	ClientRunID   string
	CharacterGUID string
	DeathInstant  string
	Ordinal       int
}

// ClientRunID returns the deterministic clientRunId digest for the given run inputs.
func ClientRunID(params RunIDParams) (string, error) {
	if err := validateASCII(params.ChallengeModeStartInstant); err != nil {
		return "", err
	}
	if err := validateCanonicalInstant(params.ChallengeModeStartInstant); err != nil {
		return "", err
	}
	if params.ChallengeMapID < 1 {
		return "", ErrInvalidChallengeMap
	}
	if params.KeystoneLevel < 0 || params.KeystoneLevel > 99 {
		return "", ErrInvalidKeystoneLevel
	}

	return contractDigest(
		domainTagRun,
		params.ChallengeModeStartInstant,
		strconv.Itoa(params.ChallengeMapID),
		strconv.Itoa(params.KeystoneLevel),
	), nil
}

// ClientEventID returns the deterministic clientEventId digest for the given event inputs.
func ClientEventID(params EventIDParams) (string, error) {
	if err := validateClientRunID(params.ClientRunID); err != nil {
		return "", err
	}
	if err := validateASCII(params.CharacterGUID); err != nil {
		return "", err
	}
	if err := validatePlayerGUID(params.CharacterGUID); err != nil {
		return "", err
	}
	if err := validateASCII(params.DeathInstant); err != nil {
		return "", err
	}
	if err := validateCanonicalInstant(params.DeathInstant); err != nil {
		return "", err
	}
	if params.Ordinal < 0 {
		return "", ErrInvalidOrdinal
	}

	return contractDigest(
		domainTagDeath,
		params.ClientRunID,
		params.CharacterGUID,
		params.DeathInstant,
		strconv.Itoa(params.Ordinal),
	), nil
}

func contractDigest(fields ...string) string {
	h := sha256.New()
	for _, field := range fields {
		b := []byte(field)
		var lenBuf [4]byte
		binary.BigEndian.PutUint32(lenBuf[:], uint32(len(b)))
		h.Write(lenBuf[:])
		h.Write(b)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func validateASCII(value string) error {
	for i := 0; i < len(value); i++ {
		if value[i] >= 0x80 {
			return fmt.Errorf("%w: byte 0x%02x at offset %d", ErrNonASCII, value[i], i)
		}
	}
	return nil
}

func validateCanonicalInstant(instant string) error {
	if !canonicalInstantPattern.MatchString(instant) {
		return ErrInvalidInstant
	}
	t, err := time.Parse("2006-01-02T15:04:05.000000000Z", instant)
	if err != nil {
		return ErrInvalidInstant
	}
	if t.Format("2006-01-02T15:04:05.000000000Z") != instant {
		return ErrInvalidInstant
	}
	return nil
}

func validateClientRunID(clientRunID string) error {
	if err := validateASCII(clientRunID); err != nil {
		return err
	}
	if !clientRunIDPattern.MatchString(clientRunID) {
		return ErrInvalidClientRunID
	}
	return nil
}

func validatePlayerGUID(guid string) error {
	if !playerGUIDPattern.MatchString(guid) {
		return ErrInvalidCharacterGUID
	}
	return nil
}
