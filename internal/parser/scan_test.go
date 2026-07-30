package parser

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

type fixedChunkReader struct {
	data []byte
	size int
}

func (r *fixedChunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := r.size
	if n > len(r.data) {
		n = len(r.data)
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

type scriptedRead struct {
	data []byte
	err  error
}

type scriptedReader struct {
	reads []scriptedRead
}

func (r *scriptedReader) Read(p []byte) (int, error) {
	if len(r.reads) == 0 {
		return 0, io.EOF
	}
	read := r.reads[0]
	r.reads = r.reads[1:]
	n := copy(p, read.data)
	return n, read.err
}

func TestScanReaderStreamsThroughHandler(t *testing.T) {
	input := strings.Join([]string{
		"COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1",
		"",
		`1/15/2026 20:06:01.0000 SYNTHETIC_UNKNOWN_EVENT,source,"Source",0xa48,0x0,dest,"Destination",0x512,0x0,tail`,
	}, "\n") + "\n"

	state := &ParserState{}
	var kinds []Kind // Bounded test collection only; ScanReader retains none.
	summary, err := ScanReader(strings.NewReader(input), DefaultMaxLineSize, state, func(event Event) error {
		kinds = append(kinds, event.Kind)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.LinesComplete != 3 || summary.EmptyLines != 1 || summary.IncompleteTail {
		t.Fatalf("summary = %#v", summary)
	}
	if len(kinds) != 3 || kinds[2] != KindUnknownEvent {
		t.Fatalf("kinds = %#v", kinds)
	}
}

func TestScanReaderKeepsRecognizedEventGenericBeforeVersionHeader(t *testing.T) {
	input := strings.Join(syntheticSpellDamage("SPELL_DAMAGE"), ",") + "\n"
	state := &ParserState{}
	var got Event
	summary, err := ScanReader(strings.NewReader(input), DefaultMaxLineSize, state, func(event Event) error {
		got = event
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != KindGeneric || got.Typed.Status != TypedStatusNotApplicable ||
		got.Typed.Payload != nil || got.Typed.Error != nil {
		t.Fatalf("event = %#v", got)
	}
	if state.Format != FormatStateNone || state.VersionHeaders != 0 {
		t.Fatalf("state = %#v", state)
	}
	if summary.TypedParsed != 0 || summary.TypedInvalid != 0 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestScanReaderHandlerErrorStopsImmediately(t *testing.T) {
	sentinel := errors.New("stop handler")
	calls := 0
	summary, err := ScanReader(strings.NewReader("a\nb\nc\n"), 10, &ParserState{}, func(Event) error {
		calls++
		if calls == 2 {
			return sentinel
		}
		return nil
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v", err)
	}
	if !errors.Is(err, ErrEventHandler) {
		t.Fatalf("error does not classify handler failure: %v", err)
	}
	if calls != 2 || summary.LinesComplete != 2 {
		t.Fatalf("calls=%d summary=%#v", calls, summary)
	}
}

func TestScanReaderRejectsZeroBytesWithoutError(t *testing.T) {
	reader := &scriptedReader{reads: []scriptedRead{{}}}
	summary, err := ScanReader(reader, DefaultMaxLineSize, &ParserState{}, func(Event) error {
		t.Fatal("handler should not be called")
		return nil
	})
	if err == nil || err.Error() != "read combat log: reader returned no data and no error" {
		t.Fatalf("error = %v", err)
	}
	if summary.LinesComplete != 0 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestScanReaderProcessesCompleteBytesBeforeReadError(t *testing.T) {
	privateErr := errors.New("private reader failure")
	reader := &scriptedReader{reads: []scriptedRead{
		{data: []byte("first\nsecond\n"), err: privateErr},
	}}
	var eventTypes []string
	summary, err := ScanReader(reader, DefaultMaxLineSize, &ParserState{}, func(event Event) error {
		eventTypes = append(eventTypes, event.EventType)
		return nil
	})
	if !errors.Is(err, ErrReadCombatLog) || !errors.Is(err, privateErr) {
		t.Fatalf("error = %v", err)
	}
	if summary.LinesComplete != 2 || len(eventTypes) != 2 ||
		eventTypes[0] != "first" || eventTypes[1] != "second" {
		t.Fatalf("summary=%#v eventTypes=%#v", summary, eventTypes)
	}
}

func TestScanReaderReadErrorIsPrivacySafe(t *testing.T) {
	privatePath := `C:\Users\PrivatePerson\WoWCombatLog-PrivateCharacter.txt`
	privateMessage := "fictional private failure"
	pathErr := &os.PathError{
		Op:   "read",
		Path: privatePath,
		Err:  errors.New(privateMessage),
	}
	reader := &scriptedReader{reads: []scriptedRead{{err: pathErr}}}
	_, err := ScanReader(reader, DefaultMaxLineSize, &ParserState{}, func(Event) error {
		return nil
	})
	if !errors.Is(err, ErrReadCombatLog) || !errors.Is(err, pathErr) {
		t.Fatalf("error classification = %v", err)
	}
	if err.Error() != ErrReadCombatLog.Error() {
		t.Fatalf("error = %q", err)
	}
	for _, sensitive := range []string{privatePath, "PrivatePerson", "PrivateCharacter", privateMessage} {
		if strings.Contains(err.Error(), sensitive) {
			t.Fatalf("error exposed %q: %q", sensitive, err.Error())
		}
	}
}

func TestScanReaderProcessesCompleteBytesWithEOF(t *testing.T) {
	reader := &scriptedReader{reads: []scriptedRead{
		{data: []byte("first\nsecond\n"), err: io.EOF},
	}}
	var eventTypes []string
	summary, err := ScanReader(reader, DefaultMaxLineSize, &ParserState{}, func(event Event) error {
		eventTypes = append(eventTypes, event.EventType)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.LinesComplete != 2 || summary.IncompleteTail || len(eventTypes) != 2 ||
		eventTypes[0] != "first" || eventTypes[1] != "second" {
		t.Fatalf("summary=%#v eventTypes=%#v", summary, eventTypes)
	}
}

func TestScanReaderIncompleteTailIsNotError(t *testing.T) {
	summary, err := ScanReader(strings.NewReader("complete\npartial"), 20, &ParserState{}, func(Event) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !summary.IncompleteTail || summary.IncompleteTailBytes != len("partial") || summary.LinesComplete != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestScanReaderContinuesAfterMalformedCSV(t *testing.T) {
	input := "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1\n" +
		"EVENT,\"unterminated\n" +
		"SYNTHETIC_UNKNOWN_EVENT,a,b,c,d,e,f,g,h\n"
	var kinds []Kind
	_, err := ScanReader(strings.NewReader(input), DefaultMaxLineSize, &ParserState{}, func(event Event) error {
		kinds = append(kinds, event.Kind)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(kinds) != 3 || kinds[1] != KindMalformed || kinds[2] != KindUnknownEvent {
		t.Fatalf("kinds = %#v", kinds)
	}
}

func TestScanReaderParsesTypedEventAcrossChunkBoundaries(t *testing.T) {
	input := "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1\n" +
		strings.Join(syntheticSpellDamage("SPELL_DAMAGE"), ",") + "\n"
	reader := &fixedChunkReader{data: []byte(input), size: 7}
	var typed TypedResult
	summary, err := ScanReader(reader, DefaultMaxLineSize, &ParserState{}, func(event Event) error {
		if event.EventType == "SPELL_DAMAGE" {
			typed = event.Typed
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TypedParsed != 1 || typed.Status != TypedStatusParsed || typed.Payload == nil {
		t.Fatalf("summary=%#v typed=%#v", summary, typed)
	}
}

func TestScanReaderOversizedTypedLineIsTerminal(t *testing.T) {
	input := "COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1\n" +
		strings.Join(syntheticSpellDamage("SPELL_DAMAGE"), ",") + "\n"
	summary, err := ScanReader(strings.NewReader(input), 100, &ParserState{}, func(Event) error {
		return nil
	})
	if !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("error = %v", err)
	}
	if summary.LinesComplete != 1 || summary.TypedParsed != 0 {
		t.Fatalf("summary = %#v", summary)
	}
}
