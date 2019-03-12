// restic transport for netbackup
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// restic transport for netbackup

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
	resticCmd = "restic"
)

// ResticTransport is the main structure for the restic transport.
type ResticTransport struct {
	Transport
}

// NewResticTransport creates a new Transport object for restic.
func NewResticTransport(config *config.Config, ex execute.Executor, log *logger.Logger, dryRun bool) (*ResticTransport, error) {
	t := &ResticTransport{}
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

// checkConfig performs restic specific checks in the configuration.
func (r *ResticTransport) checkConfig() error {
	// Source and dest directories must be set.
	// Restic works with the concept of a repository, so we don't
	// accept SourceHost or DestHost
	switch {
	case r.config.SourceDir == "":
		return fmt.Errorf("config error: SourceDir is empty")
	case r.config.DestDir == "":
		return fmt.Errorf("config error: DestDir is empty")
	case len(r.config.Exclude) != 0:
		return fmt.Errorf("config error: Exclude is not supported by restic transport")
	case r.config.SourceHost != "" || r.config.DestHost != "":
		return fmt.Errorf("config error: Cannot have source or dest host set")
	}
	return nil
}

// Run builds the command name and executes it, saving the output to the log
// file requested in the configuration or a default one if none is specified.
// Temporary files with exclusion and inclusion paths are generated, if needed,
// and removed at the end of execution. If dryRun is set, just output the
// command to be executed and the contents of the exclusion and inclusion lists
// to stderr.
func (r *ResticTransport) Run() error {
	// Create exclude list, if needed.
	err := r.createExcludeFile(r.config.Exclude)
	if err != nil {
		return err
	}
	defer os.Remove(r.excludeFile)

	// Build the full restic command line
	cmd := []string{resticCmd, "-v", "-r", r.config.SourceDir}

	if r.excludeFile != "" {
		cmd = append(cmd, fmt.Sprintf("--exclude-file=%s", r.excludeFile))
	}

	if len(r.config.ExtraArgs) != 0 {
		for _, v := range r.config.ExtraArgs {
			cmd = append(cmd, v)
		}
	}

	cmd = append(cmd, "backup", r.config.SourceDir)

	r.log.Verbosef(1, "Command: %s\n", strings.Join(cmd, " "))

	// Execute the command
	if !r.dryRun {
		return execute.RunCommand("RESTIC", cmd, r.log, r.execute, nil, nil)
	}
	return nil
}
