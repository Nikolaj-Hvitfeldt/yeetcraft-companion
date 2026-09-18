package logwatcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func writeFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func appendFile(path string, content string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}

func pollOnce(t *testing.T, watcher *Watcher, ctx context.Context) ([]string, PollResult) {
	t.Helper()
	var eventTypes []string
	result, err := watcher.Poll(ctx, func(event parser.Event) error {
		eventTypes = append(eventTypes, event.EventType)
		return nil
	})
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	return eventTypes, result
}

func TestWatcherSplitLineAcrossPollsProducesOneEvent(t *testing.T) {
	ctx := testContext(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "WoWCombatLog.txt")

	if err := writeFile(path, "line1\npart"); err != nil {
		t.Fatal(err)
	}

	identity, _, err := ResolveIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	watcher := New(path, ResetOffset(0), identity)

	first, result := pollOnce(t, watcher, ctx)
	watcher.AckCommitted(result.Committed)

	if strings.Join(first, ",") != "line1" || result.Committed.PartialLine != "part" {
		t.Fatalf("first poll events=%#v committed=%#v", first, result.Committed)
	}

	if err := appendFile(path, "ial\nline3\n"); err != nil {
		t.Fatal(err)
	}

	second, result := pollOnce(t, watcher, ctx)
	watcher.AckCommitted(result.Committed)

	if strings.Join(second, ",") != "partial,line3" {
		t.Fatalf("second poll events=%#v", second)
	}
	if result.Committed.PartialLine != "" {
		t.Fatalf("committed = %#v", result.Committed)
	}
}

func TestWatcherRestartMidFileNoDuplicateEvents(t *testing.T) {
	ctx := testContext(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "WoWCombatLog.txt")
	content := "one\ntwo\nthree\n"

	if err := writeFile(path, "one\ntw"); err != nil {
		t.Fatal(err)
	}

	identity, _, err := ResolveIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	watcher := New(path, ResetOffset(0), identity)

	first, result := pollOnce(t, watcher, ctx)
	committed := result.Committed
	if strings.Join(first, ",") != "one" || committed.PartialLine != "tw" {
		t.Fatalf("first poll events=%#v committed=%#v", first, committed)
	}

	if err := appendFile(path, "o\nthree\n"); err != nil {
		t.Fatal(err)
	}

	restarted := New(path, committed, identity)
	second, result := pollOnce(t, restarted, ctx)
	all := append(first, second...)
	if strings.Join(all, ",") != "one,two,three" {
		t.Fatalf("events=%#v", all)
	}
	if result.Committed.ByteOffset != int64(len(content)) {
		t.Fatalf("final offset=%d want %d", result.Committed.ByteOffset, len(content))
	}
}

func TestWatcherTruncateStartsNewGeneration(t *testing.T) {
	ctx := testContext(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "WoWCombatLog.txt")

	if err := writeFile(path, "first\nsecond\nthird\n"); err != nil {
		t.Fatal(err)
	}

	identity, _, err := ResolveIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	watcher := New(path, ResetOffset(0), identity)

	_, result := pollOnce(t, watcher, ctx)
	watcher.AckCommitted(result.Committed)
	if result.Committed.ByteOffset != 19 {
		t.Fatalf("offset=%d", result.Committed.ByteOffset)
	}

	if err := writeFile(path, "new\n"); err != nil {
		t.Fatal(err)
	}

	events, result := pollOnce(t, watcher, ctx)
	if !result.Truncated {
		t.Fatalf("expected truncation, result=%#v", result)
	}
	if result.Committed.Generation != 1 || result.Committed.ByteOffset != 4 {
		t.Fatalf("committed=%#v", result.Committed)
	}
	if strings.Join(events, ",") != "new" {
		t.Fatalf("events=%#v", events)
	}
}

func TestWatcherRotateToNewFilename(t *testing.T) {
	ctx := testContext(t)
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "WoWCombatLog-old.txt")
	newPath := filepath.Join(dir, "WoWCombatLog-new.txt")

	if err := writeFile(oldPath, "old\n"); err != nil {
		t.Fatal(err)
	}
	oldIdentity, _, err := ResolveIdentity(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	oldWatcher := New(oldPath, ResetOffset(0), oldIdentity)
	events, result := pollOnce(t, oldWatcher, ctx)
	oldWatcher.AckCommitted(result.Committed)
	if strings.Join(events, ",") != "old" {
		t.Fatalf("old events=%#v", events)
	}

	if err := writeFile(newPath, "fresh\n"); err != nil {
		t.Fatal(err)
	}
	newIdentity, _, err := ResolveIdentity(newPath)
	if err != nil {
		t.Fatal(err)
	}
	newWatcher := New(newPath, ResetOffset(0), newIdentity)
	events, result = pollOnce(t, newWatcher, ctx)
	if strings.Join(events, ",") != "fresh" {
		t.Fatalf("new events=%#v", events)
	}
	if result.Committed.Generation != 0 || result.Committed.ByteOffset != 6 {
		t.Fatalf("committed=%#v", result.Committed)
	}
}

func TestWatcherReplaceSamePathStartsNewGeneration(t *testing.T) {
	ctx := testContext(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "WoWCombatLog.txt")

	if err := writeFile(path, "before\npartial"); err != nil {
		t.Fatal(err)
	}
	identity, _, err := ResolveIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	watcher := New(path, ResetOffset(0), identity)

	_, result := pollOnce(t, watcher, ctx)
	watcher.AckCommitted(result.Committed)
	if result.Committed.ByteOffset != 7 || result.Committed.PartialLine != "partial" {
		t.Fatalf("committed=%#v", result.Committed)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(path, "after\n"); err != nil {
		t.Fatal(err)
	}

	events, result := pollOnce(t, watcher, ctx)
	if !result.Replaced {
		t.Fatalf("expected replacement, result=%#v", result)
	}
	if result.Committed.Generation != 1 || result.Committed.ByteOffset != 6 {
		t.Fatalf("committed=%#v", result.Committed)
	}
	if strings.Join(events, ",") != "after" {
		t.Fatalf("events=%#v", events)
	}
}

func TestWatcherConcurrentAppendWhileReading(t *testing.T) {
	ctx := testContext(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "WoWCombatLog.txt")

	if err := writeFile(path, "start\npart"); err != nil {
		t.Fatal(err)
	}
	identity, _, err := ResolveIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	watcher := New(path, ResetOffset(0), identity)

	var all []string
	var mu sync.Mutex
	appendDone := make(chan struct{})

	go func() {
		time.Sleep(10 * time.Millisecond)
		_ = appendFile(path, "ial\nend\n")
		close(appendDone)
	}()

	for {
		result, err := watcher.Poll(ctx, func(event parser.Event) error {
			mu.Lock()
			all = append(all, event.EventType)
			mu.Unlock()
			return nil
		})
		if err != nil {
			t.Fatalf("Poll: %v", err)
		}
		watcher.AckCommitted(result.Committed)
		if result.Summary.LinesComplete == 0 && !result.Summary.IncompleteTail {
			time.Sleep(5 * time.Millisecond)
			select {
			case <-appendDone:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			continue
		}
		if result.Committed.ByteOffset >= 18 && result.Committed.PartialLine == "" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if countOccurrences(all, "partial") != 1 {
		t.Fatalf("expected one partial event, got %#v", all)
	}
	if strings.Join(all, ",") != "start,partial,end" {
		t.Fatalf("events=%#v", all)
	}
}

func countOccurrences(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}
