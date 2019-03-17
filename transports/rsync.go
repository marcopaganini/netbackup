// rsync transport for netbackup
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// rsync transport for netbackup

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
	rsyncCmd = "rsync"
)

// RsyncTransport is the main structure for the rsync transport.
type RsyncTransport struct {
	Transport
}

// NewRsyncTransport creates a new Transport object for rsync.
func NewRsyncTransport(config *config.Config, ex execute.Executor, log *logger.Logger, dryRun bool) (*RsyncTransport, error) {
	t := &RsyncTransport{}
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

// checkConfig performs rsync specific checks in the configuration.
func (r *RsyncTransport) checkConfig() error {
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

// Run builds the command name and executes it, saving the output to the log
// file requested in the configuration or a default one if none is specified.
// Temporary files with exclusion and inclusion paths are generated, if needed,
// and removed at the end of execution. If dryRun is set, just output the
// command to be executed and the contents of the exclusion and inclusion lists
// to stderr.
func (r *RsyncTransport) Run() error {
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

	// Build the full rsync command line
	cmd := []string{rsyncCmd, "-avXH", "--delete", "--numeric-ids"}

	if r.excludeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--exclude-from=%s", r.excludeFile))
		cmd = append(cmd, "--delete-excluded")
	}
	if r.includeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--include-from=%s", r.includeFile))
	}
	cmd = append(cmd, r.config.ExtraArgs...)

	// In rsync, the source needs to ends with a slash or the
	// source directory will be created inside the destination.
	// The exception are the cases where the source already ends
	// in a slash (ex: /)
	src := r.buildSource(":")
	if !strings.HasSuffix(src, "/") {
		src = src + "/"
	}
	cmd = append(cmd, src)
	cmd = append(cmd, r.buildDest(":"))

	r.log.Verbosef(1, "Command: %s\n", strings.Join(cmd, " "))

	// Execute the command
	err = nil
	if !r.dryRun {
		err = execute.RunCommand("RSYNC", cmd, r.log, r.execute, nil, nil)
		if err != nil {
			rc := execute.ExitCode(err)

			// Rsync uses retcode 24 to indicate "some files disappeared during
			// the transfer" which is immaterial for our purposes. Ignore those
			// cases.
			if rc == 24 {
				r.log.Println("Note: rsync returned error 24 (some files disappeared during copy). Ignoring.")
				err = nil
			}
		}
	}
	return err
}
