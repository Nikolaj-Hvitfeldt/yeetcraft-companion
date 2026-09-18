package logwatcher

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

const DefaultChunkSize = 32 << 10

// Watcher polls a combat-log file and resumes from a committed offset.
// Polling is a hint; the committed offset is authoritative.
type Watcher struct {
	Path        string
	MaxLineSize int
	ChunkSize   int

	committed CommittedOffset
	identity  Identity
	lastInfo  os.FileInfo
}

// PollResult describes one poll read. Committed is the proposed offset after
// the poll; call AckCommitted after persistence succeeds.
type PollResult struct {
	Summary   parser.ScanSummary
	Committed CommittedOffset
	Truncated bool
	Replaced  bool
}

// New creates a watcher for path using the committed offset and identity.
func New(path string, committed CommittedOffset, identity Identity) *Watcher {
	return &Watcher{
		Path:        path,
		MaxLineSize: parser.DefaultMaxLineSize,
		ChunkSize:   DefaultChunkSize,
		committed:   committed,
		identity:    identity,
	}
}

// Committed returns the current durable offset.
func (w *Watcher) Committed() CommittedOffset {
	return w.committed
}

// Identity returns the tracked file identity.
func (w *Watcher) Identity() Identity {
	return w.identity
}

// AckCommitted records a persisted offset for subsequent polls.
func (w *Watcher) AckCommitted(committed CommittedOffset) {
	w.committed = committed
}

// Poll reads newly available bytes from the committed offset.
func (w *Watcher) Poll(ctx context.Context, handle parser.EventHandler) (PollResult, error) {
	if err := ctx.Err(); err != nil {
		return PollResult{}, err
	}
	if handle == nil {
		return PollResult{}, fmt.Errorf("poll combat log: nil event handler")
	}

	file, err := os.Open(w.Path)
	if err != nil {
		return PollResult{}, fmt.Errorf("open combat log: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return PollResult{}, fmt.Errorf("stat combat log: %w", err)
	}

	deviceInode, err := encodeFileRef(file)
	if err != nil {
		return PollResult{}, err
	}
	head, err := readHead(file, info.Size())
	if err != nil {
		return PollResult{}, err
	}
	currentIdentity := Identity{
		Path:            w.Path,
		DeviceInode:     deviceInode,
		HeadFingerprint: fingerprint(head),
	}

	result := PollResult{Committed: w.committed}
	result.Truncated = info.Size() < w.committed.ByteOffset
	result.Replaced = isReplacement(w.identity, currentIdentity, w.lastInfo, info, w.committed.ByteOffset)

	generation := w.committed.Generation
	if result.Truncated || result.Replaced {
		generation++
		w.committed = ResetOffset(generation)
		w.identity = currentIdentity
	} else if w.identity.DeviceInode == "" {
		w.identity = currentIdentity
	}

	readStart := w.committed.ByteOffset + int64(len(w.committed.PartialLine))
	if _, err := file.Seek(readStart, io.SeekStart); err != nil {
		return PollResult{}, fmt.Errorf("seek combat log: %w", err)
	}

	remaining := info.Size() - readStart
	if remaining < 0 {
		remaining = 0
	}

	limited := &limitedReader{
		r:   file,
		ctx: ctx,
		n:   remaining,
	}

	resume := w.committed.Resume()
	summary, newResume, err := parser.ScanReaderFrom(limited, w.MaxLineSize, resume, handle)
	if err != nil {
		return PollResult{}, err
	}

	result.Summary = summary
	result.Committed = FromResume(generation, newResume)
	w.lastInfo = info
	return result, nil
}

type limitedReader struct {
	r   io.Reader
	ctx context.Context
	n   int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if err := l.ctx.Err(); err != nil {
		return 0, err
	}
	if l.n <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.n {
		p = p[:l.n]
	}
	n, err := l.r.Read(p)
	l.n -= int64(n)
	if l.n == 0 && err == nil {
		return n, io.EOF
	}
	return n, err
}
