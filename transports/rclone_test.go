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

func TestRclone(t *testing.T) {
	casetests := []struct {
		name       string
		sourceDir  string
		sourceHost string
		destDir    string
		destHost   string
		transport  string
		logfile    string
		expectCmds []string
		include    []string
		exclude    []string
		dryRun     bool
		wantError  bool
	}{
		// Dry run: No command should be executed
		{
			name:      "fake",
			sourceDir: "/tmp/a",
			destDir:   "/tmp/b",
			transport: "rclone",
			logfile:   "/dev/null",
			dryRun:    true,
		},
		// Same machine (local copy)
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			transport:  "rclone",
			logfile:    "/dev/null",
			expectCmds: []string{"rclone sync -v /tmp/a /tmp/b"},
		},
		// Local source, remote destination
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			destHost:   "desthost",
			transport:  "rclone",
			logfile:    "/dev/null",
			expectCmds: []string{"rclone sync -v /tmp/a desthost:/tmp/b"},
		},
		// Remote source, local destination (unusual)
		{
			name:       "fake",
			sourceHost: "srchost",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			transport:  "rclone",
			logfile:    "/dev/null",
			expectCmds: []string{"rclone sync -v srchost:/tmp/a /tmp/b"},
		},
		// Remote source, Remote destination (server side copy)
		{
			name:       "fake",
			sourceHost: "srchost",
			sourceDir:  "/tmp/a",
			destHost:   "desthost",
			destDir:    "/tmp/b",
			transport:  "rclone",
			logfile:    "/dev/null",
			expectCmds: []string{"rclone sync -v srchost:/tmp/a desthost:/tmp/b"},
		},
		// exclude: list only
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			exclude:    []string{"x/foo", "x/bar"},
			transport:  "rclone",
			logfile:    "/dev/null",
			expectCmds: []string{"rclone sync -v --filter-from=[^ ]+ /tmp/a /tmp/b"},
		},
		// include: list only
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			include:    []string{"x/foo", "x/bar"},
			transport:  "rclone",
			logfile:    "/dev/null",
			expectCmds: []string{"rclone sync -v --filter-from=[^ ]+ /tmp/a /tmp/b"},
		},
		// include: & exclude: lists
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			exclude:    []string{"x/foo", "x/bar"},
			include:    []string{"x/foo", "x/bar"},
			transport:  "rclone",
			logfile:    "/dev/null",
			expectCmds: []string{"rclone sync -v --filter-from=[^ ]+ /tmp/a /tmp/b"},
		},
		// Test that an empty source dir results in an error
		{
			name:      "fake",
			destDir:   "/tmp/b",
			transport: "rclone",
			logfile:   "/dev/null",
			wantError: true,
		},
		// Test that an empty destination dir results in an error
		{
			name:      "fake",
			sourceDir: "/tmp/a",
			transport: "rclone",
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
			Name:       tt.name,
			SourceDir:  tt.sourceDir,
			SourceHost: tt.sourceHost,
			DestDir:    tt.destDir,
			DestHost:   tt.destHost,
			Transport:  tt.transport,
			Logfile:    tt.logfile,
			Include:    tt.include,
			Exclude:    tt.exclude,
		}

		// Create a new transport object with our fakeExecute and a sinking outLogWriter.
		rclone, err := NewRcloneTransport(cfg, fakeExecute, tt.dryRun)
		if tt.wantError && err != nil {
			continue
		}
		if err != nil {
			t.Fatalf("NewRcloneTransport failed: %v", err)
		}

		if err = rclone.Run(ctx); err != nil {
			t.Fatalf("rclone.Run failed: %v", err)
		}
		if !tt.wantError {
			if err != nil {
				t.Fatalf("Got error %q want no error", err)
			}
			match, err := reMatch(tt.expectCmds, fakeExecute.Cmds())
			if err != nil {
				t.Fatalf("Error on regexp match: %v", err)
			}
			if !match {
				t.Fatalf("command diff: Got %v, want %v", fakeExecute.Cmds(), tt.expectCmds)
			}
			continue
		}
		// Here, we want to see an error.
		if err == nil {
			t.Errorf("Got no error, want error")
		}
	}
}
