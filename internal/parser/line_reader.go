package parser

import "fmt"

// LineReader turns bounded byte chunks into complete lines.
//
// An oversized line permanently fails the reader. Callers must create a new
// LineReader; this phase deliberately does not attempt to resynchronize.
type LineReader struct {
	buf         []byte
	maxContent  int
	pendingCR   bool
	terminalErr error
}

func NewLineReader(maxContentBytes int) *LineReader {
	if maxContentBytes <= 0 {
		maxContentBytes = DefaultMaxLineSize
	}
	return &LineReader{maxContent: maxContentBytes}
}

// Write consumes chunk and returns complete lines in order. Complete lines
// preceding an oversized line are returned alongside ErrLineTooLong.
func (r *LineReader) Write(chunk []byte) ([]string, error) {
	if r.terminalErr != nil {
		return nil, r.failedError()
	}

	var lines []string
	for _, b := range chunk {
		if r.pendingCR {
			if b == '\n' {
				lines = append(lines, string(r.buf))
				r.buf = r.buf[:0]
				r.pendingCR = false
				continue
			}
			if err := r.appendContent('\r'); err != nil {
				return lines, err
			}
			r.pendingCR = false
		}

		switch b {
		case '\r':
			r.pendingCR = true
		case '\n':
			lines = append(lines, string(r.buf))
			r.buf = r.buf[:0]
		default:
			if err := r.appendContent(b); err != nil {
				return lines, err
			}
		}
	}
	return lines, nil
}

// Finalize reports an unterminated trailing line as a non-error result.
func (r *LineReader) Finalize() (string, bool, error) {
	if r.terminalErr != nil {
		return "", false, r.failedError()
	}
	if r.pendingCR {
		if err := r.appendContent('\r'); err != nil {
			return "", false, err
		}
		r.pendingCR = false
	}
	if len(r.buf) == 0 {
		return "", false, nil
	}
	return string(r.buf), true, nil
}

func (r *LineReader) Failed() bool {
	return r.terminalErr != nil
}

func (r *LineReader) BufferedContentLen() int {
	return len(r.buf)
}

func (r *LineReader) appendContent(b byte) error {
	if len(r.buf) >= r.maxContent {
		r.terminalErr = fmt.Errorf("%w: maximum %d content bytes", ErrLineTooLong, r.maxContent)
		return r.terminalErr
	}
	r.buf = append(r.buf, b)
	return nil
}

func (r *LineReader) failedError() error {
	return fmt.Errorf("%w: %w", ErrReaderTerminal, r.terminalErr)
}
