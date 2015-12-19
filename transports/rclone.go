// rclone transport for netbackup
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// rclone transport for netbackup
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"fmt"
	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/runner"
	"io"
	"os"
	"strings"
)

const (
	rcloneCmd = "rclone"
)

type RcloneTransport struct {
	Transport
}

// NewRcloneTransport creates a new Transport object for rclone.
func NewRcloneTransport(
	config *config.Config,
	runobj CommandRunner,
	outLog io.Writer,
	verbose int,
	dryRun bool) (*RcloneTransport, error) {

	t := &RcloneTransport{}
	t.config = config
	t.dryRun = dryRun
	t.outLog = outLog
	t.log = logger.New("")

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

// Run forms the command name and executes it, saving the output to the log
// file requested in the configuration or a default one if none is specified.
// Temporary files with exclusion and inclusion paths are generated, if needed,
// and removed at the end of execution. If dryRun is set, just output the
// command to be executed and the contents of the exclusion and inclusion lists
// to stderr.
func (r *RcloneTransport) Run() error {
	// Create exclude/include lists, if needed
	err := r.createExcludeFile()
	if err != nil {
		return err
	}
	defer os.Remove(r.excludeFile)

	err = r.createIncludeFile()
	if err != nil {
		return err
	}
	defer os.Remove(r.includeFile)

	// Build the full rclone command line
	cmd := []string{rcloneCmd, "sync", "-v"}

	if r.excludeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--exclude=%s", r.excludeFile))
	}
	if r.includeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--include=%s", r.includeFile))
	}
	if r.config.ExtraArgs != "" {
		cmd = append(cmd, r.config.ExtraArgs)
	}
	cmd = append(cmd, r.buildSource())
	cmd = append(cmd, r.buildDest())

	r.log.Verbosef(2, "rclone command = %q", strings.Join(cmd, " "))

	// Execute the command
	return r.runCmd(cmd)
}
