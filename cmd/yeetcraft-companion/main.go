package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/session"
)

const (
	appName    = "yeetcraft-companion"
	appVersion = "0.0.0-dev"

	exitOK      = 0
	exitFailure = 1
	exitConfig  = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr *os.File) int {
	flags := flag.NewFlagSet(appName, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s --log-file <path> [--once] [--db-path <path>] [--poll-interval <duration>]\n", appName)
		fmt.Fprintln(stderr, "Headless combat-log capture. Requires YEETCRAFT_TRACKED_GUIDS.")
		fmt.Fprintln(stderr, "Set WOW_LOG_TIMEZONE when the log uses timezone-less timestamps.")
	}

	logFile := flags.String("log-file", "", "combat-log file to watch")
	dbPath := flags.String("db-path", "", "SQLite database path (default: COMPANION_DB_PATH or ./data/companion.db)")
	once := flags.Bool("once", false, "poll once and exit (for tests)")
	pollInterval := flags.String("poll-interval", "", "poll interval for continuous mode (default 250ms)")
	inactivityTimeout := flags.String("inactivity-timeout", session.DefaultInactivityTimeout.String(), "abandon active runs after this inactivity duration")
	showVersion := flags.Bool("version", false, "print version and exit")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitFailure
	}
	if *showVersion {
		fmt.Printf("%s %s\n", appName, appVersion)
		return exitOK
	}
	if *logFile == "" || flags.NArg() != 0 {
		flags.Usage()
		return exitFailure
	}

	interval, err := parsePollInterval(*pollInterval)
	if err != nil {
		fmt.Fprintf(stderr, "poll interval: %v\n", err)
		return exitFailure
	}
	inactivity, err := parseDurationFlag(*inactivityTimeout, session.DefaultInactivityTimeout)
	if err != nil {
		fmt.Fprintf(stderr, "inactivity timeout: %v\n", err)
		return exitFailure
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return runCapture(ctx, captureOptions{
		LogFile:           *logFile,
		DBPath:            *dbPath,
		Once:              *once,
		PollInterval:      interval,
		InactivityTimeout: inactivity,
		Lookup:            os.LookupEnv,
	}, stderr)
}
