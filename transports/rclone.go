package transports

// rclone transport for netbackup
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import (
	"fmt"
	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/runner"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

const (
	rcloneCmd = "rclone"
	// DEBUG
	defaultLogDir      = "/tmp/log/netbackup"
	defaultLogDirMode  = 0770
	defaultLogFileMode = 0660
)

// RcloneTransport struct for the rclone transport.
type RcloneTransport struct {
	config *config.Config
	runner *runner.Runner
	log    *logger.Logger
	dryrun bool
}

// NewRcloneTransport creates a new Transport object for rclone.
func NewRcloneTransport(config *config.Config, runobj *runner.Runner, verbose int, dryrun bool) (*RcloneTransport, error) {
	t := &RcloneTransport{
		config: config,
		dryrun: dryrun,
		log:    logger.New("")}

	// If runner is nil, create a new one
	t.runner = runobj
	if t.runner == nil {
		t.runner = runner.New()
	}

	// Basic config checking
	if err := t.checkConfig(); err != nil {
		return nil, err
	}

	// Create a new logger with our verbosity settings
	t.log.SetVerboseLevel(verbose)
	return t, nil
}

// checkConfig performs transport specific checks in the config.
func (t *RcloneTransport) checkConfig() error {
	switch {
	case t.config.SourceDir == "":
		return fmt.Errorf("Config error: SourceDir is empty")
	case t.config.DestDir == "":
		return fmt.Errorf("Config error: DestDir is empty")
	}
	return nil
}

// writeList writes the desired list of exclusions/inclusions into a file, in a
// format suitable for this transport. The caller is responsible for deleting
// the file after use. Returns the name of the file and error.
func (t *RcloneTransport) writeList(prefix string, patterns []string) (string, error) {
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

// Run forms the command name and executes it, saving the output to the log
// file requested in the configuration or a default one if none is specified.
// Temporary files with exclusion and inclusion paths are generated, if needed,
// and removed at the end of execution. If dryrun is set, just output the
// command to be executed and the contents of the exclusion and inclusion lists
// to stderr.
func (t *RcloneTransport) Run() error {
	var (
		excludeFile string
		includeFile string
		src         string
		dst         string
		err         error
	)

	// Create exclude/include lists, if needed
	if len(t.config.ExcludeList) != 0 {
		if excludeFile, err = t.writeList("exclude", t.config.ExcludeList); err != nil {
			return err
		}
		t.log.Verbosef(3, "Exclude file %s", excludeFile)
		defer os.Remove(excludeFile)
	}

	if len(t.config.IncludeList) != 0 {
		if includeFile, err = t.writeList("include", t.config.IncludeList); err != nil {
			return err
		}
		t.log.Verbosef(3, "Include file %s", includeFile)
		defer os.Remove(includeFile)
	}

	// Construct the source & destination paths for rclone.
	// Note that rclone uses the hostname as the "storage" provider.
	// Storage providers are configured with "rclone config".
	src = t.config.SourceDir
	if t.config.SourceHost != "" {
		src = t.config.SourceHost + ":" + src
	}
	dst = t.config.DestDir
	if t.config.DestHost != "" {
		dst = t.config.DestHost + ":" + dst
	}

	// Construct the command
	cmd := []string{}
	cmd = append(cmd, rcloneCmd)
	cmd = append(cmd, "sync")
	cmd = append(cmd, "-v")

	if excludeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--exclude=%s", excludeFile))
	}
	if includeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--include=%s", includeFile))
	}
	if t.config.ExtraArgs != "" {
		cmd = append(cmd, t.config.ExtraArgs)
	}
	cmd = append(cmd, src)
	cmd = append(cmd, dst)

	t.log.Verbosef(2, "rclone command = %q", strings.Join(cmd, " "))

	err = nil
	if !t.dryrun {
		// Open logfile for append (create if needed).
		logWriter, logFile, err := createLogFile(defaultLogDir, t.config.Logfile, t.config.Name, defaultLogDirMode, defaultLogFileMode)
		if err != nil {
			return err
		}
		defer logWriter.Close()

		t.log.Verbosef(2, "Output log file: %q", logFile)
		fmt.Fprintf(logWriter, "*** Starting netbackup: %s ***\n", time.Now())
		fmt.Fprintf(logWriter, "*** Command: %s ***\n", strings.Join(cmd, " "))

		// Run
		t.runner.SetStdout(func(buf string) error { _, err := fmt.Fprintln(logWriter, buf); return err })
		t.runner.SetStderr(func(buf string) error { _, err := fmt.Fprintln(logWriter, buf); return err })
		err = t.runner.Exec(cmd)
		fmt.Fprintf(logWriter, "*** Command returned: %v ***\n", err)
	}
	return err
}
