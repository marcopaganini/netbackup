// This file is part of netbackup, a frontend to simplify periodic backups.
// For further information, check https://github.com/marcopaganini/netbackup
//
// (C) 2015-2024 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"context"
	"testing"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
)

const (
	customTestCmd = "foo --bar --baz"
)

func TestCustom(t *testing.T) {
	casetests := []struct {
		name            string
		transport       string
		logfile         string
		expectCmdsRegex []string
		dryRun          bool
		customCmd       string
		wantError       bool
	}{
		// Dry run: No command should be executed.
		{
			name:      "fake",
			transport: "custom",
			logfile:   "/dev/null",
			dryRun:    true,
			customCmd: customTestCmd,
		},
		// Same machine (local copy).
		{
			name:            "fake",
			transport:       "custom",
			logfile:         "/dev/null",
			customCmd:       customTestCmd,
			expectCmdsRegex: []string{"/bin/bash", "-c", "--", customTestCmd},
		},
		// Test that an empty customCmd results in an error.
		{
			name:      "fake",
			transport: "custom",
			logfile:   "/dev/null",
			wantError: true,
		},
	}

	for _, tt := range casetests {
		fakeExecute := NewFakeExecute()

		log := logger.New("")
		ctx := context.Background()
		ctx = logger.WithLogger(ctx, log)

		cfg := &config.Config{
			Name:      tt.name,
			Transport: tt.transport,
			Logfile:   tt.logfile,
			CustomCmd: tt.customCmd,
		}

		// Create a new custom object with our fakeExecute and a sinking outLogWriter.
		custom, err := NewCustomTransport(cfg, fakeExecute, tt.dryRun)
		if tt.wantError && err != nil {
			continue
		}
		if err != nil {
			t.Fatalf("NewCustomTransport failed: %v", err)
		}

		if err := custom.Run(ctx); err != nil {
			t.Fatalf("custom.Run failed: %v", err)
		}
		if !tt.wantError {
			if err != nil {
				t.Fatalf("Got error %q want no error", err)
			}

			match, err := reMatch(tt.expectCmdsRegex, fakeExecute.Cmds())
			if err != nil {
				t.Fatalf("Error on regexp match: %v", err)
			}
			if !match {
				t.Fatalf("command diff: Got %v, want (regex) %v", fakeExecute.Cmds(), tt.expectCmdsRegex)
			}
			continue
		}
		// Here, we want to see an error.
		if err == nil {
			t.Errorf("Got no error, want error")
		}
	}
}
