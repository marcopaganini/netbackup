// This file is part of netbackup, a frontend to simplify periodic backups.
// For further information, check https://github.com/marcopaganini/netbackup
//
// (C) 2015-2024 by Marco Paganini <paganini AT paganini DOT net>

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
	resticCmd = "restic"
)

// ResticTransport is the main structure for the restic transport.
type ResticTransport struct {
	Transport
}

// NewResticTransport creates a new Transport object for restic.
func NewResticTransport(config *config.Config, ex execute.Executor, dryRun bool) (*ResticTransport, error) {
	t := &ResticTransport{}
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
	case len(r.config.Include) != 0:
		return fmt.Errorf("config error: Include is not supported by restic transport")
	case r.config.SourceHost != "":
		return fmt.Errorf("config error: Cannot have source host set (push mode only)")
	}
	return nil
}

// Run builds the command name and executes it, saving the output to the log
// file requested in the configuration or a default one if none is specified.
// Temporary files with exclusion and inclusion paths are generated, if needed,
// and removed at the end of execution. If dryRun is set, just output the
// command to be executed and the contents of the exclusion and inclusion lists
// to stderr.
func (r *ResticTransport) Run(ctx context.Context) error {
	var (
		// Cmds contains multiple commands to be executed.
		// Failure in one command will stop the chain of executions.
		cmds [][]string

		excludeFile string
	)

	log := logger.LoggerValue(ctx)

	// Create exclude file list, if needed.
	if len(r.config.Exclude) != 0 {
		excludeFile, err := writeList(ctx, "exclude", r.config.Exclude)
		if err != nil {
			return err
		}
		defer os.Remove(excludeFile)
	}

	// Generate restic command-line.
	// restic -v -v [--exclude-file=<file>] [extra_args] --repo <destination_repo> backup <sourcedir>
	resticBin := resticCmd
	if r.config.CustomBin != "" {
		resticBin = r.config.CustomBin
	}

	cmd := strings.Split(resticBin, " ")
	cmd = append(cmd, "-v", "-v")

	if len(r.config.Exclude) != 0 {
		cmd = append(cmd, fmt.Sprintf("--exclude-file=%s", excludeFile))
	}

	cmd = append(cmd, r.config.ExtraArgs...)
	cmd = append(cmd, []string{"--repo", r.buildDest(":")}...)
	cmd = append(cmd, "backup", r.config.SourceDir)

	// Add to list of commands.
	cmds = append(cmds, cmd)

	// Create expiration command, if required.
	if r.config.ExpireDays != 0 {
		cmd = append(cmd, []string{"forget", fmt.Sprintf("--keep-within=%dd", r.config.ExpireDays), "--prune"}...)
		cmds = append(cmds, cmd)
	}

	for i, c := range cmds {
		log.Verbosef(1, "Command(%d/%d): %s\n", i+1, len(cmds), strings.Join(c, " "))
	}

	// Execute the command(s)
	if !r.dryRun {
		for _, c := range cmds {
			err := execute.RunCommand(ctx, "RESTIC", c, r.execute, nil, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
