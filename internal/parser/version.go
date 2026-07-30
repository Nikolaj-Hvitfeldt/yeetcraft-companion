package parser

import (
	"errors"
	"fmt"
	"strconv"
)

type VersionHeader struct {
	FormatVersion      int
	AdvancedLogEnabled bool
	BuildVersion       string
	ProjectID          int
}

type FormatState int

const (
	FormatStateNone FormatState = iota
	FormatStateSupportedV22
	FormatStateQuarantinedUnsupported
)

func (s FormatState) String() string {
	switch s {
	case FormatStateSupportedV22:
		return "supported"
	case FormatStateQuarantinedUnsupported:
		return "quarantined"
	default:
		return "none"
	}
}

type ParserState struct {
	Format              FormatState
	FormatVersion       int
	VersionHeaders      int
	SupportedHeaders    int
	UnsupportedHeaders  int
	UnsupportedProjects int
	MalformedHeaders    int
}

func IsVersionHeader(fields []string) bool {
	return len(fields) > 0 && fields[0] == "COMBAT_LOG_VERSION"
}

func ParseVersionHeader(fields []string) (VersionHeader, error) {
	var header VersionHeader
	if len(fields) != 8 {
		return header, fmt.Errorf("parse version header: expected 8 fields, got %d", len(fields))
	}
	expectedKeys := []string{"COMBAT_LOG_VERSION", "ADVANCED_LOG_ENABLED", "BUILD_VERSION", "PROJECT_ID"}
	for i, key := range expectedKeys {
		if fields[i*2] != key {
			return header, fmt.Errorf("parse version header: expected key %q at field %d", key, i*2)
		}
	}

	formatVersion, err := strconv.Atoi(fields[1])
	if err != nil {
		return header, fmt.Errorf("parse version header format version: invalid integer")
	}
	header.FormatVersion = formatVersion
	advancedMarker, err := strconv.Atoi(fields[3])
	if err != nil || (advancedMarker != 0 && advancedMarker != 1) {
		return header, fmt.Errorf("parse version header advanced marker: expected 0 or 1")
	}
	header.AdvancedLogEnabled = advancedMarker == 1
	header.BuildVersion = fields[5]
	projectID, err := strconv.Atoi(fields[7])
	if err != nil {
		return header, fmt.Errorf("parse version header project ID: invalid integer")
	}
	header.ProjectID = projectID
	if formatVersion != SupportedFormatVersion {
		return header, fmt.Errorf("%w: version %d", ErrUnsupportedFormatVersion, formatVersion)
	}
	if projectID != 1 {
		return header, fmt.Errorf("%w: project ID %d", ErrUnsupportedProject, projectID)
	}
	return header, nil
}

func (s *ParserState) applyVersion(header VersionHeader, parseErr error) {
	s.VersionHeaders++
	if parseErr == nil {
		s.SupportedHeaders++
		if s.Format != FormatStateQuarantinedUnsupported {
			s.Format = FormatStateSupportedV22
			s.FormatVersion = SupportedFormatVersion
		}
		return
	}

	switch {
	case errors.Is(parseErr, ErrUnsupportedFormatVersion):
		s.UnsupportedHeaders++
	case errors.Is(parseErr, ErrUnsupportedProject):
		s.UnsupportedProjects++
	default:
		s.MalformedHeaders++
	}
	s.Format = FormatStateQuarantinedUnsupported
	s.FormatVersion = 0
}
