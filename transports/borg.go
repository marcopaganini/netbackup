// This file is part of netbackup, a frontend to simplify periodic backups.
// For further information, check https://github.com/marcopaganini/netbackup
//
// (C) 2015-2026 by Marco Paganini <paganini AT paganini DOT net>

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
	borgCmd = "borg"
)

// BorgTransport is the main structure for the borg transport.
type BorgTransport struct {
	Transport
}

// NewBorgTransport creates a new Transport object for borg.
func NewBorgTransport(config *config.Config, ex execute.Executor, dryRun bool) (*BorgTransport, error) {
	t := &BorgTransport{}
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

// checkConfig performs borg specific checks in the configuration.
func (b *BorgTransport) checkConfig() error {
	// Source and dest directories must be set.
	// Borg works with the concept of a repository, so we don't
	// accept SourceHost (borg runs on the machine holding the source).
	switch {
	case b.config.SourceDir == "":
		return fmt.Errorf("config error: SourceDir is empty")
	case b.config.DestDir == "":
		return fmt.Errorf("config error: DestDir is empty")
	case b.config.SourceHost != "":
		return fmt.Errorf("config error: Cannot have source host set (push mode only)")
	}
	return nil
}

// Run builds the command name and executes it, saving the output to the log
// file requested in the configuration or a default one if none is specified.
// Temporary files with exclusion paths are generated, if needed, and removed at
// the end of execution. If dryRun is set, just output the command to be
// executed and the contents of the exclusion list to stderr.
func (b *BorgTransport) Run(ctx context.Context) error {
	var (
		// Cmds contains multiple commands to be executed.
		// Failure in one command will stop the chain of executions.
		cmds [][]string
	)

	log := logger.LoggerValue(ctx)

	// Borg reads its passphrase from the BORG_PASSPHRASE environment
	// variable. If a keyfile is configured, populate it from the file
	// so the same keyfile can be shared with the LUKS code path.
	if b.config.Keyfile != "" {
		data, rerr := os.ReadFile(b.config.Keyfile)
		if rerr != nil {
			return fmt.Errorf("error reading keyfile %q: %v", b.config.Keyfile, rerr)
		}
		if serr := os.Setenv("BORG_PASSPHRASE", string(data)); serr != nil {
			return fmt.Errorf("error setting BORG_PASSPHRASE: %v", serr)
		}
	}

	// Build the borg patterns file (passed via --patterns-from) from the
	// include and exclude lists. Borg treats "R <pattern>" as a recursive
	// include and "- <pattern>" as an exclude; patterns are processed in
	// order, with the first match winning. When includes are present we
	// append a catch-all exclusion ("- **") so that "include" means "only
	// these" (matching netbackup's documented convention for the other
	// transports). Without that catch-all, borg would ignore include
	// patterns and back up everything.
	var filterPatterns []string
	for _, v := range b.config.Include {
		filterPatterns = append(filterPatterns, "R "+v)
	}
	for _, v := range b.config.Exclude {
		filterPatterns = append(filterPatterns, "- "+v)
	}
	if len(b.config.Include) != 0 {
		filterPatterns = append(filterPatterns, "- **")
	}

	var filterFile string
	if len(filterPatterns) != 0 {
		var err error
		filterFile, err = writeList(ctx, "patterns", filterPatterns)
		if err != nil {
			return err
		}
		defer os.Remove(filterFile)
	}

	// Generate borg command-line.
	// borg create [--patterns-from=<file>] [extra_args] <repo>::<archive> <sourcedir>
	borgBin := borgCmd
	if b.config.CustomBin != "" {
		borgBin = b.config.CustomBin
	}

	cmd := strings.Split(borgBin, " ")
	cmd = append(cmd, "create", "--filter=AME", "--verbose", "--list", "--stats")

	if len(filterPatterns) != 0 {
		cmd = append(cmd, fmt.Sprintf("--patterns-from=%s", filterFile))
	}

	cmd = append(cmd, b.config.ExtraArgs...)

	// Borg archive names must be unique; we use the backup name plus the
	// borg "{now}" placeholder (expanded by borg at runtime) so that each
	// invocation creates a new archive. We also tag the repo with the backup
	// name as a prefix so pruning only affects archives created by this backup.
	archive := fmt.Sprintf("%s::%s-{now}", b.buildDest(":"), b.config.Name)
	cmd = append(cmd, archive)
	cmd = append(cmd, b.config.SourceDir)

	// Add to list of commands.
	cmds = append(cmds, cmd)

	// Create prune and compact commands, if required.
	if b.config.ExpireDays != 0 {
		prune := strings.Split(borgBin, " ")
		prune = append(prune, "prune",
			fmt.Sprintf("--keep-within=%dd", b.config.ExpireDays),
			fmt.Sprintf("--glob=%s", b.config.Name),
			b.buildDest(":"))
		cmds = append(cmds, prune)

		compact := strings.Split(borgBin, " ")
		compact = append(compact, "compact", b.buildDest(":"))
		cmds = append(cmds, compact)
	}

	for i, c := range cmds {
		log.Verbosef(1, "Command(%d/%d): %s\n", i+1, len(cmds), strings.Join(c, " "))
	}

	// Execute the command(s)
	if !b.dryRun {
		for _, c := range cmds {
			err := execute.RunCommand(ctx, "BORG", c, b.execute, nil, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
