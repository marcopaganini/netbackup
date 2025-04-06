// This file is part of netbackup, a frontend to simplify periodic backups.
// For further information, check https://github.com/marcopaganini/netbackup
//
// (C) 2015-2024 by Marco Paganini <paganini AT paganini DOT net>

// Package transports handles all transports for netbackup (the actual programs
// that transfer data).
package transports

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/execute"
)

// Transport represents all transports
type Transport struct {
	config  *config.Config
	execute execute.Executor
	dryRun  bool
}

// writeList writes the desired list of exclusions/inclusions into a file, in a
// format suitable for this transport. The caller is responsible for deleting
// the file after use. Returns the name of the file and error.
func writeList(ctx context.Context, prefix string, patterns []string) (string, error) {
	var w *os.File
	var err error
	log := logger.LoggerValue(ctx)

	if w, err = os.CreateTemp("/tmp", prefix); err != nil {
		return "", fmt.Errorf("error creating pattern file for %s list: %v", prefix, err)
	}
	defer w.Close()
	for _, v := range patterns {
		fmt.Fprintln(w, v)
	}

	log.Verbosef(3, "Contents of %q file:\n", prefix)
	_ = displayFile(ctx, w.Name())
	return w.Name(), nil
}

// displayFile opens the specified file and output all lines in it using the
// log object.
func displayFile(ctx context.Context, fname string) error {
	r, err := os.Open(fname)
	if err != nil {
		return fmt.Errorf("error opening %q: %v", fname, err)
	}
	defer r.Close()

	log := logger.LoggerValue(ctx)
	s := bufio.NewScanner(r)
	for s.Scan() {
		log.Verboseln(3, s.Text())
	}
	return nil
}

// checkConfig performs basic checks in the configuration.
func (t *Transport) checkConfig() error {
	switch {
	case t.config.SourceDir == "":
		return fmt.Errorf("config error: SourceDir is empty")
	case t.config.DestDir == "":
		return fmt.Errorf("config error: DestDir is empty")
	}
	return nil
}

// createFilterFile creates a filter file, in the rsync/rclone style, with the
// include and exclude patterns and returns the filename.
func (t *Transport) createFilterFile(ctx context.Context, include, exclude []string) (string, error) {
	log := logger.LoggerValue(ctx)

	if len(include) == 0 && len(exclude) == 0 {
		return "", nil
	}
	// Create filter list.
	var filter []string
	for _, v := range include {
		filter = append(filter, "+ "+v)
	}
	for _, v := range exclude {
		filter = append(filter, "- "+v)
	}

	fname, err := writeList(ctx, "filter", filter)
	if err != nil {
		return "", err
	}
	log.Verbosef(2, "Filter file: %q\n", fname)
	// Display file contents to log if dryRun mode
	if t.dryRun {
		_ = displayFile(ctx, fname)
	}
	return fname, nil
}

// buildSource creates the backup source based on the source host and path.
// The default is [sourcehost<separator>]sourcepath. The default separator
// is ":".
func (t *Transport) buildSource(separator string) string {
	src := t.config.SourceDir
	if t.config.SourceHost != "" {
		src = t.config.SourceHost + separator + src
	}
	return src
}

// buildDest creates the backup destination based on the destination host and
// path.  The default is [desthost:<separator>]destpath. The default separator
// is ":".
func (t *Transport) buildDest(separator string) string {
	dst := t.config.DestDir
	if t.config.DestHost != "" {
		dst = t.config.DestHost + separator + dst
	}
	return dst
}

// Run forms the command name and executes it, saving the output to the log
// file requested in the configuration or a default one if none is specified.
// Temporary files with exclusion and inclusion paths are generated, if needed,
// and removed at the end of execution. If dryRun is set, just output the
// command to be executed and the contents of the exclusion and inclusion lists
// to stderr. Note that this is the generic form which only outputs an error.
// It needs to be overridden to something useful in structs that embed the
// Transport structure.
func (t *Transport) Run(_ context.Context) error {
	return fmt.Errorf("internal error: Attempted to execute generic Run method")
}
