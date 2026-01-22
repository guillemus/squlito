package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"squlito/internal/app"
)

func main() {
	programName := filepath.Base(os.Args[0])
	dbPath, showHelp, err := parseArgs(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		printUsage(os.Stderr, programName)
		os.Exit(2)
	}

	if showHelp {
		printUsage(os.Stdout, programName)
		return
	}

	err = app.Run(dbPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseArgs(args []string) (string, bool, error) {
	flags := flag.NewFlagSet("squlito", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}

	err := flags.Parse(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return "", true, nil
		}
		return "", false, err
	}

	remaining := flags.Args()
	if len(remaining) == 0 {
		return "", true, nil
	}

	if len(remaining) > 1 {
		return "", false, fmt.Errorf("expected 1 database argument, got %d", len(remaining))
	}

	return remaining[0], false, nil
}

func printUsage(writer io.Writer, programName string) {
	_, _ = fmt.Fprintf(writer, "Usage:\n  %s [--help] <database>\n\n", programName)
	_, _ = fmt.Fprintln(writer, "Arguments:")
	_, _ = fmt.Fprintln(writer, "  database  path to a SQLite database file")
	_, _ = fmt.Fprintln(writer, "\nFlags:")
	_, _ = fmt.Fprintln(writer, "  --help    show this help message")
}
