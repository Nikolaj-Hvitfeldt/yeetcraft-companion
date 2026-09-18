package parser

import (
	"fmt"
	"io"
)

const scanChunkSize = 32 << 10

type EventHandler func(Event) error

// ResumeState captures parser progress for incremental reads.
type ResumeState struct {
	ParserState *ParserState
	PendingLine string
	LineNumber  int
	ByteOffset  int64
}

type ScanSummary struct {
	LinesComplete       int
	EmptyLines          int
	IncompleteTail      bool
	IncompleteTailBytes int
	BytesConsumed       int64
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
	resume := ResumeState{ParserState: state}
	summary, _, err := ScanReaderFrom(reader, maxLineSize, resume, handle)
	return summary, err
}

// ScanReaderFrom resumes parsing from resume and returns updated resume state.
// The reader must be positioned at resume.ByteOffset. When resume.PendingLine
// is non-empty, that content is applied before reading from the reader.
// BytesConsumed counts only bytes that completed full lines, including line
// terminators. Incomplete trailing content is returned on resume.PendingLine.
func ScanReaderFrom(
	reader io.Reader,
	maxLineSize int,
	resume ResumeState,
	handle EventHandler,
) (ScanSummary, ResumeState, error) {
	var summary ScanSummary
	if reader == nil {
		return summary, resume, fmt.Errorf("scan combat log: nil reader")
	}
	if handle == nil {
		return summary, resume, fmt.Errorf("scan combat log: nil event handler")
	}

	state := resume.ParserState
	if state == nil {
		state = &ParserState{}
		resume.ParserState = state
	}

	lineReader := NewLineReader(maxLineSize)
	initialPending := resume.PendingLine
	if initialPending != "" {
		if _, err := lineReader.Write([]byte(initialPending)); err != nil {
			return summary, resume, &LineError{Op: "resume", LineNumber: resume.LineNumber + 1, Err: err}
		}
	}

	counting := &countingReader{reader: reader}
	chunk := make([]byte, scanChunkSize)
	lineNumber := resume.LineNumber
	fileStart := resume.ByteOffset

	for {
		n, readErr := counting.Read(chunk)
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
					tail := pendingTail(lineReader)
					summary.BytesConsumed = counting.bytes
					resume.LineNumber = lineNumber
					resume.ByteOffset = nextByteOffset(fileStart, initialPending, counting.bytes, tail)
					resume.PendingLine = tail
					return summary, resume, fmt.Errorf("%w at combat log line %d: %w", ErrEventHandler, lineNumber, err)
				}
			}
			if lineErr != nil {
				summary.BytesConsumed = counting.bytes
				resume.LineNumber = lineNumber
				resume.ByteOffset = nextByteOffset(fileStart, initialPending, counting.bytes, "")
				resume.PendingLine = ""
				return summary, resume, &LineError{Op: "read", LineNumber: lineNumber + 1, Err: lineErr}
			}
		}

		if readErr != nil {
			if readErr != io.EOF {
				summary.BytesConsumed = counting.bytes
				resume.LineNumber = lineNumber
				resume.ByteOffset = nextByteOffset(fileStart, initialPending, counting.bytes, "")
				resume.PendingLine = ""
				return summary, resume, &scanReadError{cause: readErr}
			}
			tail, hasTail, err := lineReader.Finalize()
			if err != nil {
				summary.BytesConsumed = counting.bytes
				resume.LineNumber = lineNumber
				resume.ByteOffset = nextByteOffset(fileStart, initialPending, counting.bytes, "")
				resume.PendingLine = ""
				return summary, resume, &LineError{Op: "finalize", LineNumber: lineNumber + 1, Err: err}
			}
			summary.BytesConsumed = counting.bytes - int64(len(tail))
			summary.IncompleteTail = hasTail
			summary.IncompleteTailBytes = len(tail)
			resume.LineNumber = lineNumber
			resume.ByteOffset = nextByteOffset(fileStart, initialPending, counting.bytes, tail)
			resume.PendingLine = tail
			return summary, resume, nil
		}
		if n == 0 {
			return summary, resume, fmt.Errorf("read combat log: reader returned no data and no error")
		}
	}
}

type countingReader struct {
	reader io.Reader
	bytes  int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.bytes += int64(n)
	return n, err
}

func nextByteOffset(fileStart int64, initialPending string, readBytes int64, tail string) int64 {
	return fileStart + int64(len(initialPending)) + readBytes - int64(len(tail))
}

func pendingTail(lineReader *LineReader) string {
	if lineReader.BufferedContentLen() == 0 {
		return ""
	}
	tail, hasTail, err := lineReader.Finalize()
	if err != nil || !hasTail {
		return ""
	}
	return tail
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
