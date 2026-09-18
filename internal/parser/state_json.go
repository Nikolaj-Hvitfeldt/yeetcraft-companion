package parser

import (
	"encoding/json"
	"fmt"
)

type parserStateJSON struct {
	Format              int `json:"format"`
	FormatVersion       int `json:"formatVersion"`
	VersionHeaders      int `json:"versionHeaders"`
	SupportedHeaders    int `json:"supportedHeaders"`
	UnsupportedHeaders  int `json:"unsupportedHeaders"`
	UnsupportedProjects int `json:"unsupportedProjects"`
	MalformedHeaders    int `json:"malformedHeaders"`
}

// EncodeParserState serializes parser quarantine state for SQLite resume.
func EncodeParserState(state *ParserState) (string, error) {
	if state == nil {
		return "{}", nil
	}
	data, err := json.Marshal(parserStateJSON{
		Format:              int(state.Format),
		FormatVersion:       state.FormatVersion,
		VersionHeaders:      state.VersionHeaders,
		SupportedHeaders:    state.SupportedHeaders,
		UnsupportedHeaders:  state.UnsupportedHeaders,
		UnsupportedProjects: state.UnsupportedProjects,
		MalformedHeaders:    state.MalformedHeaders,
	})
	if err != nil {
		return "", fmt.Errorf("encode parser state: %w", err)
	}
	return string(data), nil
}

// DecodeParserState restores parser quarantine state from SQLite resume JSON.
func DecodeParserState(raw string) (*ParserState, error) {
	if raw == "" || raw == "{}" {
		return &ParserState{}, nil
	}
	var dto parserStateJSON
	if err := json.Unmarshal([]byte(raw), &dto); err != nil {
		return nil, fmt.Errorf("decode parser state: %w", err)
	}
	return &ParserState{
		Format:              FormatState(dto.Format),
		FormatVersion:       dto.FormatVersion,
		VersionHeaders:      dto.VersionHeaders,
		SupportedHeaders:    dto.SupportedHeaders,
		UnsupportedHeaders:  dto.UnsupportedHeaders,
		UnsupportedProjects: dto.UnsupportedProjects,
		MalformedHeaders:    dto.MalformedHeaders,
	}, nil
}
