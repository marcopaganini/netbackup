// This file is part of netbackup, a frontend to simplify periodic backups.
// For further information, check https://github.com/marcopaganini/netbackup
//
// (C) 2015-2026 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"context"
	"os"
	"testing"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
)

func TestBorg(t *testing.T) {
	casetests := []struct {
		name       string
		sourceDir  string
		sourceHost string
		destDir    string
		destHost   string
		transport  string
		logfile    string
		expireDays int
		expectCmds []string
		include    []string
		exclude    []string
		dryRun     bool
		wantError  bool
	}{
		// Dry run: No command should be executed.
		{
			name:      "fake",
			sourceDir: "/tmp/a",
			destDir:   "/tmp/b",
			transport: "borg",
			logfile:   "/dev/null",
			dryRun:    true,
		},
		// Same machine (local copy).
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			transport:  "borg",
			logfile:    "/dev/null",
			expectCmds: []string{"borg", "create", "--filter=AME", "--verbose", "--list", "--stats", "/tmp/b::fake-{now}", "/tmp/a"},
		},

		// Local source, remote destination.
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			destHost:   "desthost",
			transport:  "borg",
			logfile:    "/dev/null",
			expectCmds: []string{"borg", "create", "--filter=AME", "--verbose", "--list", "--stats", "desthost:/tmp/b::fake-{now}", "/tmp/a"},
		},

		// Remote source, local destination (error, unsupported).
		{
			name:       "fake",
			sourceHost: "srchost",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			transport:  "borg",
			logfile:    "/dev/null",
			wantError:  true,
		},

		// Remote source, Remote destination (server side copy, not supported by borg.)
		{
			name:       "fake",
			sourceHost: "srchost",
			sourceDir:  "/tmp/a",
			destHost:   "desthost",
			destDir:    "/tmp/b",
			transport:  "borg",
			logfile:    "/dev/null",
			wantError:  true,
		},

		// Exclude list.
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			exclude:    []string{"x/foo", "x/bar"},
			transport:  "borg",
			logfile:    "/dev/null",
			expectCmds: []string{"borg", "create", "--filter=AME", "--verbose", "--list", "--stats", "--patterns-from=[^ ]+", "/tmp/b::fake-{now}", "/tmp/a"},
		},

		// Include list (only these paths are backed up).
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			include:    []string{"x/foo"},
			transport:  "borg",
			logfile:    "/dev/null",
			expectCmds: []string{"borg", "create", "--filter=AME", "--verbose", "--list", "--stats", "--patterns-from=[^ ]+", "/tmp/b::fake-{now}", "/tmp/a"},
		},

		// Include and exclude lists combined.
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			include:    []string{"x/foo"},
			exclude:    []string{"x/bar"},
			transport:  "borg",
			logfile:    "/dev/null",
			expectCmds: []string{"borg", "create", "--filter=AME", "--verbose", "--list", "--stats", "--patterns-from=[^ ]+", "/tmp/b::fake-{now}", "/tmp/a"},
		},

		// Expire days: create + prune + compact.
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			expireDays: 10,
			transport:  "borg",
			logfile:    "/dev/null",
			expectCmds: []string{
				"borg", "create", "--filter=AME", "--verbose", "--list", "--stats", "/tmp/b::fake-{now}", "/tmp/a",
				"borg", "prune", "--keep-within=10d", "--glob=fake", "/tmp/b",
				"borg", "compact", "/tmp/b",
			},
		},

		// Test that an empty source dir results in error.
		{
			name:      "fake",
			destDir:   "/tmp/b",
			transport: "borg",
			logfile:   "/dev/null",
			wantError: true,
		},

		// Test that an empty destination dir results in error.
		{
			name:      "fake",
			sourceDir: "/tmp/a",
			transport: "borg",
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
			ExpireDays: tt.expireDays,
			Include:    tt.include,
			Exclude:    tt.exclude,
		}

		// Create a new borg object with our fakeExecute and a sinking outLogWriter.
		borg, err := NewBorgTransport(cfg, fakeExecute, tt.dryRun)
		if tt.wantError && err != nil {
			continue
		}
		if err != nil {
			t.Fatalf("NewBorgTransport failed: %v", err)
		}

		if err := borg.Run(ctx); err != nil {
			t.Fatalf("borg.Run failed: %v", err)
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

// TestBorgKeyfile verifies that a configured keyfile is used to set
// the BORG_PASSPHRASE environment variable (so the same keyfile can be
// shared with the LUKS code path).
func TestBorgKeyfile(t *testing.T) {
	log := logger.New("")
	ctx := context.Background()
	ctx = logger.WithLogger(ctx, log)

	// Create a temporary keyfile.
	tmp, err := os.CreateTemp("", "borgkey")
	if err != nil {
		t.Fatalf("unable to create temp keyfile: %v", err)
	}
	defer os.Remove(tmp.Name())
	content := "super-secret-passphrase\n"
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("unable to write keyfile: %v", err)
	}
	tmp.Close()

	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Transport: "borg",
		Logfile:   "/dev/null",
		Keyfile:   tmp.Name(),
	}

	borg, err := NewBorgTransport(cfg, NewFakeExecute(), false)
	if err != nil {
		t.Fatalf("NewBorgTransport failed: %v", err)
	}
	if err := borg.Run(ctx); err != nil {
		t.Fatalf("borg.Run failed: %v", err)
	}
	defer os.Unsetenv("BORG_PASSPHRASE")

	got := os.Getenv("BORG_PASSPHRASE")
	if got != content {
		t.Fatalf("BORG_PASSPHRASE = %q, want %q", got, content)
	}
}
