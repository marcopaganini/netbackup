// This file is part of netbackup, a frontend to simplify periodic backups.
// For further information, check https://github.com/marcopaganini/netbackup
//
// (C) 2015-2025 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"context"
	"fmt"
	"strings"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/execute"
)

// CustomTransport is the main structure for the custom transport.
type CustomTransport struct {
	Transport
}

// NewCustomTransport creates a new Transport object for the custom transport.
func NewCustomTransport(config *config.Config, ex execute.Executor, dryRun bool) (*CustomTransport, error) {
	t := &CustomTransport{}
	t.config = config
	t.dryRun = dryRun

	// Create a new executor if needed.
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

// checkConfig performs custom specific checks in the configuration.
func (r *CustomTransport) checkConfig() error {
	// Make sure custom command is defined.
	if r.config.CustomCmd == "" {
		return fmt.Errorf("config error: CustomCmd is empty")
	}
	return nil
}

// Run executes the command specified in config.CustomCmd, saving the output to
// the log file requested in the configuration or a default one if none is
// specified.  Temporary files with exclusion and inclusion paths are
// generated, if needed, and removed at the end of execution. If dryRun is set,
// just output the command to be executed to stderr.
func (r *CustomTransport) Run(ctx context.Context) error {
	log := logger.LoggerValue(ctx)

	// CustomCmd is run with the default shell.
	cmd := execute.WithShell(r.config.CustomCmd)
	log.Verbosef(1, "Command: %s\n", strings.Join(cmd, " "))

	if r.dryRun {
		return nil
	}

	return execute.RunCommand(ctx, "CUSTOM", cmd, r.execute, nil, nil)
}
