// rdiff-backup transport for netbackup
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
//
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"fmt"
	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/execute"
	"io"
	"os"
	"strings"
	"time"
)

const (
	rdiffBackupCmd = "rdiff-backup"
)

// RdiffBackupTransport is the main structure for the rdiff-backup transport.
type RdiffBackupTransport struct {
	Transport
}

// NewRdiffBackupTransport creates a new Transport object for rdiff-backup.
func NewRdiffBackupTransport(
	config *config.Config,
	ex Executor,
	outLog io.Writer,
	dryRun bool) (*RdiffBackupTransport, error) {

	t := &RdiffBackupTransport{}
	t.config = config
	t.dryRun = dryRun
	t.outLog = outLog
	t.log = logger.New("")

	// If execute object is nil, create a new one
	t.execute = ex
	if t.execute == nil {
		t.execute = execute.New()
	}

	// Basic config checking
	if err := t.checkConfig(); err != nil {
		return nil, err
	}

	// Create a new logger with our verbosity settings
	return t, nil
}

// checkConfig performs rdiff-backup specific checks in the configuration.
func (r *RdiffBackupTransport) checkConfig() error {
	// Source and dest directories must be set.
	// Either source host or destination host can be set, not both.
	switch {
	case r.config.SourceDir == "":
		return fmt.Errorf("Config error: SourceDir is empty")
	case r.config.DestDir == "":
		return fmt.Errorf("Config error: DestDir is empty")
	case r.config.SourceHost != "" && r.config.DestHost != "":
		return fmt.Errorf("Config error: Cannot have source & dest host set")
	}
	return nil
}

// matchSlice returns true if the string s matches any substring within
// the passed slice, false otherwise.
func matchSlice(slice []string, s string) bool {
	for _, v := range slice {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
}

// runCmd runs rdiff-backup and exclude a number of spammy messages from
// stdout/err.  This is the rdiff-specific version of runCmd.
func (r *RdiffBackupTransport) runCmd(cmd []string) error {
	if r.dryRun {
		return nil
	}

	spam := []string{
		"POSIX ACLs not supported",
		"Unable to import win32security module",
		"not supported by filesystem at",
		"escape_dos_devices not required by filesystem at",
		"Reading globbing filelist",
		"Updated mirror temp file.* does not match source",
		"/.gvfs"}

	// Print stdout and stderr excluding any messages that match
	// a substring in our spam list.
	p := func(buf string) error {
		if !matchSlice(spam, buf) {
			_, err := fmt.Fprintln(r.outLog, buf)
			return err
		}
		return nil
	}

	// Log
	r.log.Verbosef(2, "*** Command = %q", strings.Join(cmd, " "))
	fmt.Fprintf(r.outLog, "*** Command: %q\n", strings.Join(cmd, " "))

	// Run
	r.execute.SetStdout(p)
	r.execute.SetStderr(p)
	err := r.execute.Exec(cmd)
	fmt.Fprintf(r.outLog, "*** Command returned: %v ***\n", err)
	return err
}

// Run forms the command name and executes it, saving the output to the log
// file requested in the configuration or a default one if none is specified.
// Temporary files with exclusion and inclusion paths are generated, if needed,
// and removed at the end of execution. If dryRun is set, just output the
// command to be executed and the contents of the exclusion and inclusion lists
// to stderr.
func (r *RdiffBackupTransport) Run() error {
	// Create exclude/include lists, if needed
	err := r.createExcludeFile(absPaths(r.config.SourceDir, r.config.Exclude))
	if err != nil {
		return err
	}
	defer os.Remove(r.excludeFile)

	err = r.createIncludeFile(absPaths(r.config.SourceDir, r.config.Include))
	if err != nil {
		return err
	}
	defer os.Remove(r.includeFile)

	// Build the full rclone command line
	cmd := []string{
		rdiffBackupCmd,
		"--verbosity=5",
		"--terminal-verbosity=5",
		"--preserve-numerical-ids",
		"--exclude-sockets",
		"--exclude-other-filesystems",
		"--force"}

	if r.excludeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--exclude-globbing-filelist=%s", r.excludeFile))
	}
	if r.includeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--include-globbing-filelist=%s", r.includeFile))
	}
	if r.config.ExtraArgs != "" {
		cmd = append(cmd, r.config.ExtraArgs)
	}

	// rdiff-backup uses double colons as host/destination separators.
	src := r.config.SourceDir
	if r.config.SourceHost != "" {
		src = r.config.SourceHost + "::" + src
	}
	dst := r.config.DestDir
	if r.config.DestHost != "" {
		dst = r.config.DestHost + "::" + dst
	}
	cmd = append(cmd, src)
	cmd = append(cmd, dst)

	fmt.Fprintf(r.outLog, "*** Starting netbackup: %s\n", time.Now())

	// Execute the command
	err = r.runCmd(cmd)
	if err != nil {
		return err
	}

	// Remove older files, if requested.
	if r.config.RdiffBackupMaxAge != 0 {
		cmd := []string{
			rdiffBackupCmd,
			fmt.Sprintf("--remove-older-than=%dD", r.config.RdiffBackupMaxAge),
			"--force",
			dst}
		r.log.Verbosef(2, "rdiff-backup command = %q", strings.Join(cmd, " "))
		fmt.Fprintf(r.outLog, "*** Starting netbackup: %s\n", time.Now())
		return r.runCmd(cmd)
	}

	return nil

}
