// transports.go: Common routines for transports
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"bufio"
	"fmt"
	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/execute"
	"io/ioutil"
	"os"
)

// Transport represents all transports
type Transport struct {
	config      *config.Config
	execute     execute.Executor
	log         *logger.Logger
	dryRun      bool
	excludeFile string
	includeFile string
}

// writeList writes the desired list of exclusions/inclusions into a file, in a
// format suitable for this transport. The caller is responsible for deleting
// the file after use. Returns the name of the file and error.
func writeList(prefix string, patterns []string) (string, error) {
	var w *os.File
	var err error

	if w, err = ioutil.TempFile("/tmp", prefix); err != nil {
		return "", fmt.Errorf("Error creating pattern file for %s list: %v", prefix, err)
	}
	defer w.Close()
	for _, v := range patterns {
		fmt.Fprintln(w, v)
	}
	return w.Name(), nil
}

// displayFile opens the specified file and output all lines in it using the
// log object.
func displayFile(log *logger.Logger, fname string) error {
	r, err := os.Open(fname)
	if err != nil {
		return fmt.Errorf("error opening %q: %v", fname, err)
	}
	defer r.Close()

	log.Verbosef(3, "Contents of %q:\n", fname)
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
		return fmt.Errorf("Config error: SourceDir is empty")
	case t.config.DestDir == "":
		return fmt.Errorf("Config error: DestDir is empty")
	}
	return nil
}

// createExcludeFile creates a file with the list of patterns to be excluded.
// The file is only created if config.Exclude is set.
func (t *Transport) createExcludeFile(paths []string) error {
	if len(t.config.Exclude) == 0 {
		return nil
	}
	fname, err := writeList("exclude", paths)
	if err != nil {
		return err
	}
	t.log.Verbosef(2, "Exclude file: %q\n", fname)
	// Display file contents to log if dryRun mode
	if t.dryRun {
		displayFile(t.log, fname)
	}
	t.excludeFile = fname

	return nil
}

// createIncludeFile creates a file with the list of patterns to be included.
// The file is only created if config.Include is set.
func (t *Transport) createIncludeFile(paths []string) error {
	if len(t.config.Include) == 0 {
		return nil
	}
	fname, err := writeList("include", paths)
	if err != nil {
		return err
	}
	t.log.Verbosef(2, "Include file: %q\n", fname)
	// Display file contents to log if dryRun mode
	if t.dryRun {
		displayFile(t.log, fname)
	}
	t.includeFile = fname
	return nil
}

// buildSource creates the backup source based on the source host and path.
// The default is [sourcehost:]sourcepath
func (t *Transport) buildSource() string {
	src := t.config.SourceDir
	if t.config.SourceHost != "" {
		src = t.config.SourceHost + ":" + src
	}
	return src
}

// buildDest creates the backup destinatino based on the destination host and
// path.  The default is [desthost:]destpath.
func (t *Transport) buildDest() string {
	dst := t.config.DestDir
	if t.config.DestHost != "" {
		dst = t.config.DestHost + ":" + dst
	}
	return dst
}

// Run forms the command name and executes it, saving the output to the log
// file requested in the configuration or a default one if none is specified.
// Temporary files with exclusion and inclusion paths are generated, if needed,
// and removed at the end of execution. If dryRun is set, just output the
// command to be executed and the contents of the exclusion and inclusion lists
// to stderr. Note that this is the generic form which only outputs an error.
// It needs to be overriden to something useful in structs that embed the
// Transport structure.
func (t *Transport) Run() error {
	return fmt.Errorf("internal error: Attempted to execute generic Run method.")
}
