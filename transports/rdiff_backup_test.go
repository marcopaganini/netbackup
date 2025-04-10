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

func TestRdiffBackup(t *testing.T) {
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
		expireDays int
		dryRun     bool
		wantError  bool
	}{
		// Dry run: No command should be executed
		{
			name:      "fake",
			sourceDir: "/tmp/a",
			destDir:   "/tmp/b",
			transport: "rdiff-backup",
			logfile:   "/dev/null",
			dryRun:    true,
		},
		// Same machine (local copy)
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			transport:  "rdiff-backup",
			logfile:    "/dev/null",
			expectCmds: []string{"rdiff-backup", "--verbosity=5", "--terminal-verbosity=5", "--preserve-numerical-ids", "--exclude-sockets", "--force", "/tmp/a", "/tmp/b"},
		},
		// Local source, remote destination
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			destHost:   "desthost",
			transport:  "rdiff-backup",
			logfile:    "/dev/null",
			expectCmds: []string{"rdiff-backup", "--verbosity=5", "--terminal-verbosity=5", "--preserve-numerical-ids", "--exclude-sockets", "--force", "/tmp/a", "desthost::/tmp/b"},
		},
		// Remote source, local destination (unusual)
		{
			name:       "fake",
			sourceHost: "srchost",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			transport:  "rdiff-backup",
			logfile:    "/dev/null",
			expectCmds: []string{"rdiff-backup", "--verbosity=5", "--terminal-verbosity=5", "--preserve-numerical-ids", "--exclude-sockets", "--force", "srchost::/tmp/a", "/tmp/b"},
		},
		// Remote source, Remote destination (server side copy) not supported under
		// rdiff-backup and should return an error.
		{
			name:       "fake",
			sourceHost: "srchost",
			sourceDir:  "/tmp/a",
			destHost:   "desthost",
			destDir:    "/tmp/b",
			transport:  "rdiff-backup",
			logfile:    "/dev/null",
			wantError:  true,
		},
		// Exclude list only
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			exclude:    []string{"x/foo", "x/bar"},
			transport:  "rdiff-backup",
			logfile:    "/dev/null",
			expectCmds: []string{"rdiff-backup", "--verbosity=5", "--terminal-verbosity=5", "--preserve-numerical-ids", "--exclude-sockets", "--force", "--exclude-globbing-filelist=[^ ]+", "/tmp/a", "/tmp/b"},
		},
		// Include list only
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			include:    []string{"x/foo", "x/bar"},
			transport:  "rdiff-backup",
			logfile:    "/dev/null",
			expectCmds: []string{"rdiff-backup", "--verbosity=5", "--terminal-verbosity=5", "--preserve-numerical-ids", "--exclude-sockets", "--force", "--include-globbing-filelist=[^ ]+", "/tmp/a", "/tmp/b"},
		},
		// Include & Exclude lists
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			exclude:    []string{"x/foo", "x/bar"},
			include:    []string{"x/foo", "x/bar"},
			transport:  "rdiff-backup",
			logfile:    "/dev/null",
			expectCmds: []string{"rdiff-backup", "--verbosity=5", "--terminal-verbosity=5", "--preserve-numerical-ids", "--exclude-sockets", "--force", "--exclude-globbing-filelist=[^ ]+", "--include-globbing-filelist=[^ ]+", "/tmp/a", "/tmp/b"},
		},
		// Expiration.
		// TODO: fix this test
		//{
		//    name:       "fake",
		//    sourceDir:  "/tmp/a",
		//    destDir:    "/tmp/b",
		//    transport:  "rdiff-backup",
		//    logfile:    "/dev/null",
		//    expireDays: 7,
		//    expectCmds: []string{
		//        {"rdiff-backup", "--verbosity=5", "--terminal-verbosity=5", "--preserve-numerical-ids", "--exclude-sockets", "--force", "/tmp/a", "/tmp/b"},
		//        {"rdiff-backup", "--remove-older-than=7D", "--force /tmp/b"},
		//    },
		//},
		// Test that an empty source dir results in an error
		{
			name:      "fake",
			destDir:   "/tmp/b",
			transport: "rdiff-backup",
			logfile:   "/dev/null",
			wantError: true,
		},
		// Test that an empty destination dir results in an error
		{
			name:      "fake",
			sourceDir: "/tmp/a",
			transport: "rdiff-backup",
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
			ExpireDays: tt.expireDays,
			Logfile:    tt.logfile,
			Include:    tt.include,
			Exclude:    tt.exclude,
		}

		// Create a new transport object with our fakeExecute and a sinking outLogWriter.
		rdiffBackup, err := NewRdiffBackupTransport(cfg, fakeExecute, tt.dryRun)
		if tt.wantError && err != nil {
			continue
		}
		if err != nil {
			t.Fatalf("NewRdiffBackupTransport failed: %v", err)
		}

		if err := rdiffBackup.Run(ctx); err != nil {
			t.Fatalf("rdiffBackup.Run failed: %v", err)
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
