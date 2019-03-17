// rdiff-backup transport for netbackup
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
//
// (C) 2015-2018 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"fmt"
	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/execute"
	"os"
	"strings"
)

const (
	rdiffBackupCmd = "rdiff-backup"
)

// RdiffBackupTransport is the main structure for the rdiff-backup transport.
type RdiffBackupTransport struct {
	Transport
}

// NewRdiffBackupTransport creates a new Transport object for rdiff-backup.
func NewRdiffBackupTransport(config *config.Config, ex execute.Executor, log *logger.Logger, dryRun bool) (*RdiffBackupTransport, error) {
	t := &RdiffBackupTransport{}
	t.config = config
	t.log = log
	t.dryRun = dryRun

	// If execute object is nil, create a new one
	t.execute = ex
	if t.execute == nil {
		t.execute = execute.New()
	}

	// Basic config checking
	if err := t.checkConfig(); err != nil {
		return nil, err
	}
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

// Run forms the command name and executes it, saving the output to the log
// file requested in the configuration or a default one if none is specified.
// Temporary files with exclusion and inclusion paths are generated, if needed,
// and removed at the end of execution. If dryRun is set, just output the
// command to be executed and the contents of the exclusion and inclusion lists
// to stderr.
func (r *RdiffBackupTransport) Run() error {
	// Create exclude/include lists, if needed
	err := r.createExcludeFile(r.config.Exclude)
	if err != nil {
		return err
	}
	defer os.Remove(r.excludeFile)

	err = r.createIncludeFile(r.config.Include)
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
		"--force"}

	if r.excludeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--exclude-globbing-filelist=%s", r.excludeFile))
	}
	if r.includeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--include-globbing-filelist=%s", r.includeFile))
	}
	cmd = append(cmd, r.config.ExtraArgs...)

	// rdiff-backup uses double colons as host/destination separators.
	cmd = append(cmd, r.buildSource("::"))
	cmd = append(cmd, r.buildDest("::"))

	// Execute the command
	spam := []string{
		"POSIX ACLs not supported",
		"Unable to import win32security module",
		"not supported by filesystem at",
		"escape_dos_devices not required by filesystem at",
		"Reading globbing filelist",
		"Updated mirror temp file.* does not match source",
		"/.gvfs"}

	r.log.Verbosef(1, "Command: %s\n", strings.Join(cmd, " "))

	if !r.dryRun {
		// Run
		err = execute.RunCommand("RDIFF-BACKUP", cmd, r.log, r.execute, spam, spam)
		if err != nil {
			return err
		}
	}

	// Remove older files, if requested.
	if r.config.RdiffBackupMaxAge != 0 {
		cmd := []string{
			rdiffBackupCmd,
			fmt.Sprintf("--remove-older-than=%dD", r.config.RdiffBackupMaxAge),
			"--force",
			r.buildDest("::")}

		r.log.Verbosef(1, "Command: %s\n", strings.Join(cmd, " "))
		if !r.dryRun {
			return execute.RunCommand("RDIFF-BACKUP", cmd, r.log, r.execute, spam, spam)
		}
	}

	return nil

}
