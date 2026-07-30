package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

const (
	exitOK          = 0
	exitFailure     = 1
	exitUnsupported = 2
	exitIncomplete  = 3
)

type eventCounts struct {
	commonHeader     int
	metadata         int
	unknown          int
	generic          int
	malformedCSV     int
	malformedVersion int
	malformedCommon  int
	malformedOther   int
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("logprobe", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: logprobe --file <path>")
		fmt.Fprintln(stderr, "Reads a combat log and prints privacy-safe parser counts.")
	}
	filePath := flags.String("file", "", "combat-log file to read")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitFailure
	}
	if *filePath == "" || flags.NArg() != 0 {
		flags.Usage()
		return exitFailure
	}

	file, err := os.Open(*filePath)
	if err != nil {
		fmt.Fprintf(stderr, "open combat log: %v\n", err)
		return exitFailure
	}
	defer file.Close()

	state := &parser.ParserState{}
	var counts eventCounts
	summary, err := parser.ScanReader(file, parser.DefaultMaxLineSize, state, func(event parser.Event) error {
		switch event.Kind {
		case parser.KindCommonHeader:
			counts.commonHeader++
		case parser.KindMetadata:
			counts.metadata++
		case parser.KindUnknownEvent:
			counts.unknown++
		case parser.KindGeneric:
			counts.generic++
		case parser.KindMalformed:
			switch event.Malformed {
			case parser.MalformedCSV:
				counts.malformedCSV++
			case parser.MalformedVersionHeader:
				counts.malformedVersion++
			case parser.MalformedCommonHeader:
				counts.malformedCommon++
			default:
				counts.malformedOther++
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(stderr, "scan combat log: %v\n", err)
		return exitFailure
	}

	repeats := state.VersionHeaders - 1
	if repeats < 0 {
		repeats = 0
	}
	fmt.Fprintf(stdout, "lines_complete: %d\n", summary.LinesComplete)
	fmt.Fprintf(stdout, "empty_lines: %d\n", summary.EmptyLines)
	fmt.Fprintf(stdout, "version_headers_supported: %d\n", state.SupportedHeaders)
	fmt.Fprintf(stdout, "version_headers_unsupported: %d\n", state.UnsupportedHeaders)
	fmt.Fprintf(stdout, "version_headers_unsupported_project: %d\n", state.UnsupportedProjects)
	fmt.Fprintf(stdout, "version_header_repeats: %d\n", repeats)
	fmt.Fprintf(stdout, "events_common_header: %d\n", counts.commonHeader)
	fmt.Fprintf(stdout, "events_metadata: %d\n", counts.metadata)
	fmt.Fprintf(stdout, "unknown_event_types: %d\n", counts.unknown)
	fmt.Fprintf(stdout, "events_generic: %d\n", counts.generic)
	fmt.Fprintf(stdout, "malformed_csv: %d\n", counts.malformedCSV)
	fmt.Fprintf(stdout, "malformed_version_headers: %d\n", counts.malformedVersion)
	fmt.Fprintf(stdout, "malformed_common_headers: %d\n", counts.malformedCommon)
	fmt.Fprintf(stdout, "malformed_other_records: %d\n", counts.malformedOther)
	fmt.Fprintf(stdout, "incomplete_trailing: %d\n", boolInt(summary.IncompleteTail))
	fmt.Fprintf(stdout, "incomplete_trailing_bytes: %d\n", summary.IncompleteTailBytes)
	fmt.Fprintf(stdout, "format_state: %s\n", state.Format)

	if state.Format != parser.FormatStateSupportedV22 {
		return exitUnsupported
	}
	if summary.IncompleteTail {
		return exitIncomplete
	}
	return exitOK
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
