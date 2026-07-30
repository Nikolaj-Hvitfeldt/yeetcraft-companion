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
	TypedParsed         int
	TypedInvalid        int
	TypedErrors         TypedErrorSummary
	Diagnostics         ValidationDiagnosticSummary
}

type TypedErrorSummary struct {
	FieldCount    int
	EmptyRequired int
	Integer       int
	Hex           int
	Float         int
	Boolean       int
}

type ValidationDiagnosticSummary struct {
	Total                       int
	AdvancedInfoGUIDMismatch    int
	EnvironmentalSourceNotZero  int
	SwingSchoolUnexpected       int
	AbilityHintUnknown          int
	EnvironmentalTypeUnknown    int
	AdvancedUnknownFieldNonZero int
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
				recordTypedSummary(&summary, event.Typed)
				if err := handle(event); err != nil {
					return summary, fmt.Errorf("%w at combat log line %d: %w", ErrEventHandler, lineNumber, err)
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

func recordTypedSummary(summary *ScanSummary, typed TypedResult) {
	switch typed.Status {
	case TypedStatusParsed:
		summary.TypedParsed++
	case TypedStatusInvalid:
		summary.TypedInvalid++
		if typed.Error == nil {
			return
		}
		switch typed.Error.Kind {
		case TypedErrorFieldCount:
			summary.TypedErrors.FieldCount++
		case TypedErrorEmptyRequired:
			summary.TypedErrors.EmptyRequired++
		case TypedErrorInteger:
			summary.TypedErrors.Integer++
		case TypedErrorHex:
			summary.TypedErrors.Hex++
		case TypedErrorFloat:
			summary.TypedErrors.Float++
		case TypedErrorBoolean:
			summary.TypedErrors.Boolean++
		}
	}

	for _, diagnostic := range []struct {
		flag    ValidationDiagnostics
		counter *int
	}{
		{DiagnosticAdvancedInfoGUIDMismatch, &summary.Diagnostics.AdvancedInfoGUIDMismatch},
		{DiagnosticEnvironmentalSourceNotZero, &summary.Diagnostics.EnvironmentalSourceNotZero},
		{DiagnosticSwingSchoolUnexpected, &summary.Diagnostics.SwingSchoolUnexpected},
		{DiagnosticAbilityHintUnknown, &summary.Diagnostics.AbilityHintUnknown},
		{DiagnosticEnvironmentalTypeUnknown, &summary.Diagnostics.EnvironmentalTypeUnknown},
		{DiagnosticAdvancedUnknownFieldNonZero, &summary.Diagnostics.AdvancedUnknownFieldNonZero},
	} {
		if typed.Diagnostics.Has(diagnostic.flag) {
			(*diagnostic.counter)++
			summary.Diagnostics.Total++
		}
	}
}
