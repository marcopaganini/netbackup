// transports.go: Common routines for transports
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"fmt"
	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/execute"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// Executor defines the interface used to run commands.
type Executor interface {
	SetStdout(execute.CallbackFunc)
	SetStderr(execute.CallbackFunc)
	Exec([]string) error
}

// Transport represents all transports
type Transport struct {
	config      *config.Config
	execute     Executor
	outLog      io.Writer
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
// The file is only created if config.Include is set.
func (t *Transport) createExcludeFile() error {
	var (
		fname string
		err   error
	)

	if len(t.config.Exclude) != 0 {
		if fname, err = writeList("exclude", t.config.Exclude); err != nil {
			return err
		}
		t.log.Verbosef(3, "Exclude file: %q", fname)
		t.excludeFile = fname
	}
	return nil
}

// createIncludeFile creates a file with the list of patterns to be included,
// The file is only created if config.Include is set.
func (t *Transport) createIncludeFile() error {
	var (
		fname string
		err   error
	)
	if len(t.config.Include) != 0 {
		if fname, err = writeList("include", t.config.Include); err != nil {
			return err
		}
		t.log.Verbosef(3, "Include file: %q", fname)
		t.includeFile = fname
	}
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
	return fmt.Errorf("Internal error: Attempted to execute generic Run method.")
}

// runCmd executes the command.
func (t *Transport) runCmd(cmd []string) error {
	var err error
	err = nil
	if !t.dryRun {
		fmt.Fprintf(t.outLog, "*** Starting netbackup: %s ***\n", time.Now())
		fmt.Fprintf(t.outLog, "*** Command: %s ***\n", strings.Join(cmd, " "))

		// Run
		t.execute.SetStdout(func(buf string) error { _, err := fmt.Fprintln(t.outLog, buf); return err })
		t.execute.SetStderr(func(buf string) error { _, err := fmt.Fprintln(t.outLog, buf); return err })
		err = t.execute.Exec(cmd)
		fmt.Fprintf(t.outLog, "*** Command returned: %v ***\n", err)
	}
	return err
}
