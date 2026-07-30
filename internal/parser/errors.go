package parser

import (
	"errors"
	"fmt"
)

const (
	// DefaultMaxLineSize is the maximum number of content bytes in a line.
	// LF and CRLF terminators are not included in the limit.
	DefaultMaxLineSize = 256 << 10

	// SupportedFormatVersion is the combat-log format supported by this parser.
	SupportedFormatVersion = 22
)

var (
	ErrLineTooLong              = errors.New("combat log line exceeds maximum size")
	ErrReaderTerminal           = errors.New("line reader is in a terminal failed state")
	ErrUnsupportedFormatVersion = errors.New("unsupported combat log format version")
	ErrUnsupportedProject       = errors.New("unsupported combat log project")
	ErrMalformedCSV             = errors.New("malformed combat log CSV")
	ErrEventHandler             = errors.New("combat log event handler failed")
)

// LineError adds an operation and line number without exposing log content.
type LineError struct {
	Op         string
	LineNumber int
	Err        error
}

func (e *LineError) Error() string {
	if e.LineNumber > 0 {
		return fmt.Sprintf("%s combat log line %d: %v", e.Op, e.LineNumber, e.Err)
	}
	return fmt.Sprintf("%s combat log: %v", e.Op, e.Err)
}

func (e *LineError) Unwrap() error {
	return e.Err
}
