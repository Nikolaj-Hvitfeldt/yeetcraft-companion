package parser

import (
	"errors"
	"strings"
	"testing"
)

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
	if calls != 2 || summary.LinesComplete != 2 {
		t.Fatalf("calls=%d summary=%#v", calls, summary)
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
