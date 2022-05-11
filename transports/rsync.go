// rsync transport for netbackup
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// rsync transport for netbackup

package transports

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/execute"
)

const (
	rsyncCmd = "rsync"
)

// RsyncTransport is the main structure for the rsync transport.
type RsyncTransport struct {
	Transport
}

// NewRsyncTransport creates a new Transport object for rsync.
func NewRsyncTransport(config *config.Config, ex execute.Executor, dryRun bool) (*RsyncTransport, error) {
	t := &RsyncTransport{}
	t.config = config
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
func (r *RsyncTransport) Run(ctx context.Context) error {
	log := logger.LoggerValue(ctx)

	// Build the full rsync command line
	cmd := []string{rsyncCmd}
	if r.config.CustomBin != "" {
		cmd = strings.Split(r.config.CustomBin, " ")
	}
	cmd = append(cmd, "-avAXH", "--delete", "--numeric-ids")

	// Create filter file, if needed.
	if len(r.config.Include) > 0 || len(r.config.Exclude) > 0 {
		filterFile, err := r.createFilterFile(ctx, r.config.Include, r.config.Exclude)
		if err != nil {
			return err
		}
		defer os.Remove(filterFile)
		// Merge the filter file in the filter specification.
		cmd = append(cmd, fmt.Sprintf("--filter=merge %s", filterFile))
	}
	if len(r.config.Exclude) > 0 {
		cmd = append(cmd, "--delete-excluded")
	}
	cmd = append(cmd, r.config.ExtraArgs...)

	// In rsync, the source needs to ends with a slash or the source directory
	// will be created inside the destination.  The exception are the cases
	// where the source already ends in a slash (ex: /)
	src := r.buildSource(":")
	if !strings.HasSuffix(src, "/") {
		src = src + "/"
	}
	cmd = append(cmd, src)
	cmd = append(cmd, r.buildDest(":"))

	log.Verbosef(1, "Command: %s\n", strings.Join(cmd, " "))

	if r.dryRun {
		return nil
	}

	// Execute the command
	err := execute.RunCommand(ctx, "RSYNC", cmd, r.execute, nil, nil)
	if err != nil {
		// Rsync uses retcode 24 to indicate "some files disappeared during
		// the transfer" which is immaterial for our purposes. Ignore those
		// cases.
		rc := execute.ExitCode(err)
		if rc == 24 {
			err = nil
		}
	}
	return err
}
