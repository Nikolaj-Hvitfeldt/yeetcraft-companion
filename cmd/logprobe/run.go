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
	typed            typedEventCounts
}

type parsedInvalidCounts struct {
	parsed  int
	invalid int
}

type typedEventCounts struct {
	spellDamage      parsedInvalidCounts
	rangeDamage      parsedInvalidCounts
	swingDamage      parsedInvalidCounts
	environmental    parsedInvalidCounts
	encounterStart   parsedInvalidCounts
	encounterEnd     parsedInvalidCounts
	challengeModeEnd parsedInvalidCounts
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
		fmt.Fprintln(stderr, "open combat log: unable to open requested file")
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
		counts.typed.record(event)
		return nil
	})
	if err != nil {
		writeScanError(stderr, err)
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
	fmt.Fprintf(stdout, "typed_payloads_parsed: %d\n", summary.TypedParsed)
	fmt.Fprintf(stdout, "typed_payloads_invalid: %d\n", summary.TypedInvalid)
	fmt.Fprintf(stdout, "typed_error_field_count: %d\n", summary.TypedErrors.FieldCount)
	fmt.Fprintf(stdout, "typed_error_empty_required: %d\n", summary.TypedErrors.EmptyRequired)
	fmt.Fprintf(stdout, "typed_error_integer: %d\n", summary.TypedErrors.Integer)
	fmt.Fprintf(stdout, "typed_error_hex: %d\n", summary.TypedErrors.Hex)
	fmt.Fprintf(stdout, "typed_error_float: %d\n", summary.TypedErrors.Float)
	fmt.Fprintf(stdout, "typed_error_boolean: %d\n", summary.TypedErrors.Boolean)
	printTypedEventCounts(stdout, "spell_damage", counts.typed.spellDamage)
	printTypedEventCounts(stdout, "range_damage", counts.typed.rangeDamage)
	printTypedEventCounts(stdout, "swing_damage", counts.typed.swingDamage)
	printTypedEventCounts(stdout, "environmental_damage", counts.typed.environmental)
	printTypedEventCounts(stdout, "encounter_start", counts.typed.encounterStart)
	printTypedEventCounts(stdout, "encounter_end", counts.typed.encounterEnd)
	printTypedEventCounts(stdout, "challenge_mode_end", counts.typed.challengeModeEnd)
	fmt.Fprintf(stdout, "validation_diagnostics: %d\n", summary.Diagnostics.Total)
	fmt.Fprintf(stdout, "validation_advanced_info_guid_mismatch: %d\n", summary.Diagnostics.AdvancedInfoGUIDMismatch)
	fmt.Fprintf(stdout, "validation_environmental_source_not_zero: %d\n", summary.Diagnostics.EnvironmentalSourceNotZero)
	fmt.Fprintf(stdout, "validation_swing_school_unexpected: %d\n", summary.Diagnostics.SwingSchoolUnexpected)
	fmt.Fprintf(stdout, "validation_ability_hint_unknown: %d\n", summary.Diagnostics.AbilityHintUnknown)
	fmt.Fprintf(stdout, "validation_environmental_type_unknown: %d\n", summary.Diagnostics.EnvironmentalTypeUnknown)
	fmt.Fprintf(stdout, "validation_advanced_unknown_field_non_zero: %d\n", summary.Diagnostics.AdvancedUnknownFieldNonZero)
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

func writeScanError(writer io.Writer, err error) {
	fmt.Fprintln(writer, classifyScanError(err))
}

func classifyScanError(err error) string {
	switch {
	case errors.Is(err, parser.ErrLineTooLong):
		return "scan combat log: oversized line"
	case errors.Is(err, parser.ErrEventHandler):
		return "scan combat log: event handler failed"
	default:
		return "scan combat log: read or parse failure"
	}
}

func (counts *typedEventCounts) record(event parser.Event) {
	var target *parsedInvalidCounts
	switch event.EventType {
	case "SPELL_DAMAGE":
		target = &counts.spellDamage
	case "RANGE_DAMAGE":
		target = &counts.rangeDamage
	case "SWING_DAMAGE":
		target = &counts.swingDamage
	case "ENVIRONMENTAL_DAMAGE":
		target = &counts.environmental
	case "ENCOUNTER_START":
		target = &counts.encounterStart
	case "ENCOUNTER_END":
		target = &counts.encounterEnd
	case "CHALLENGE_MODE_END":
		target = &counts.challengeModeEnd
	default:
		return
	}
	switch event.Typed.Status {
	case parser.TypedStatusParsed:
		target.parsed++
	case parser.TypedStatusInvalid:
		target.invalid++
	}
}

func printTypedEventCounts(writer io.Writer, eventName string, counts parsedInvalidCounts) {
	fmt.Fprintf(writer, "typed_%s_parsed: %d\n", eventName, counts.parsed)
	fmt.Fprintf(writer, "typed_%s_invalid: %d\n", eventName, counts.invalid)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
