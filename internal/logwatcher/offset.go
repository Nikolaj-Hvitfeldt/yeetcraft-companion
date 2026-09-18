package logwatcher

import (
	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

// CommittedOffset is the durable read position. Advance it only after events
// and parser state derived from the consumed bytes are persisted.
type CommittedOffset struct {
	Generation  int64
	ByteOffset  int64
	PartialLine string
	LineNumber  int
	ParserState *parser.ParserState
}

// Resume converts the committed offset into parser resume state.
func (c CommittedOffset) Resume() parser.ResumeState {
	return parser.ResumeState{
		ParserState: c.ParserState,
		PendingLine: c.PartialLine,
		LineNumber:  c.LineNumber,
		ByteOffset:  c.ByteOffset,
	}
}

// FromResume maps parser resume state into a committed offset for generation.
func FromResume(generation int64, resume parser.ResumeState) CommittedOffset {
	return CommittedOffset{
		Generation:  generation,
		ByteOffset:  resume.ByteOffset,
		PartialLine: resume.PendingLine,
		LineNumber:  resume.LineNumber,
		ParserState: resume.ParserState,
	}
}

// Reset returns a zeroed offset for the given generation.
func ResetOffset(generation int64) CommittedOffset {
	return CommittedOffset{Generation: generation}
}
