package parser

import (
	"fmt"
	"io"
)

const scanChunkSize = 32 << 10

type EventHandler func(Event) error

type ScanSummary struct {
	LinesComplete       int
	EmptyLines          int
	IncompleteTail      bool
	IncompleteTailBytes int
}

// ScanReader parses complete lines incrementally and never retains all events.
// A handler error stops scanning immediately and is returned with the partial
// summary.
func ScanReader(
	reader io.Reader,
	maxLineSize int,
	state *ParserState,
	handle EventHandler,
) (ScanSummary, error) {
	var summary ScanSummary
	if reader == nil {
		return summary, fmt.Errorf("scan combat log: nil reader")
	}
	if handle == nil {
		return summary, fmt.Errorf("scan combat log: nil event handler")
	}
	if state == nil {
		state = &ParserState{}
	}

	lineReader := NewLineReader(maxLineSize)
	chunk := make([]byte, scanChunkSize)
	lineNumber := 0

	for {
		n, readErr := reader.Read(chunk)
		if n > 0 {
			lines, lineErr := lineReader.Write(chunk[:n])
			for _, line := range lines {
				lineNumber++
				summary.LinesComplete++
				event := ParseLine(lineNumber, line, state)
				if event.Kind == KindEmpty {
					summary.EmptyLines++
				}
				if err := handle(event); err != nil {
					return summary, fmt.Errorf("handle combat log line %d: %w", lineNumber, err)
				}
			}
			if lineErr != nil {
				return summary, &LineError{Op: "read", LineNumber: lineNumber + 1, Err: lineErr}
			}
		}

		if readErr != nil {
			if readErr != io.EOF {
				return summary, fmt.Errorf("read combat log: %w", readErr)
			}
			tail, hasTail, err := lineReader.Finalize()
			if err != nil {
				return summary, &LineError{Op: "finalize", LineNumber: lineNumber + 1, Err: err}
			}
			summary.IncompleteTail = hasTail
			summary.IncompleteTailBytes = len(tail)
			return summary, nil
		}
		if n == 0 {
			return summary, fmt.Errorf("read combat log: reader returned no data and no error")
		}
	}
}
