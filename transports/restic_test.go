// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015-2018 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"testing"
)

func TestRestic(t *testing.T) {
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
		// Dry run: No command should be executed.
		{
			name:      "fake",
			sourceDir: "/tmp/a",
			destDir:   "/tmp/b",
			transport: "restic",
			logfile:   "/dev/null",
			dryRun:    true,
		},
		// Same machine (local copy).
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			transport:  "restic",
			logfile:    "/dev/null",
			expectCmds: []string{"restic -v -r /tmp/b backup /tmp/a"},
		},

		// Local source, remote destination.
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			destHost:   "desthost",
			transport:  "restic",
			logfile:    "/dev/null",
			expectCmds: []string{"restic -v -r desthost:/tmp/b backup /tmp/a"},
		},

		// Remote source, local destination (error, unsupported).
		{
			name:       "fake",
			sourceHost: "srchost",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			transport:  "restic",
			logfile:    "/dev/null",
			wantError:  true,
		},

		// Remote source, Remote destination (server side copy, not supported by restic.)
		{
			name:       "fake",
			sourceHost: "srchost",
			sourceDir:  "/tmp/a",
			destHost:   "desthost",
			destDir:    "/tmp/b",
			transport:  "restic",
			logfile:    "/dev/null",
			wantError:  true,
		},

		// Exclude list.
		{
			name:       "fake",
			sourceDir:  "/tmp/a",
			destDir:    "/tmp/b",
			exclude:    []string{"x/foo", "x/bar"},
			transport:  "restic",
			logfile:    "/dev/null",
			expectCmds: []string{"restic -v --exclude-file=[^ ]* -r /tmp/b backup /tmp/a"},
		},

		// Test that an empty source dir results in error.
		{
			name:      "fake",
			destDir:   "/tmp/b",
			transport: "restic",
			logfile:   "/dev/null",
			wantError: true,
		},

		// Test that an empty destination dir results in error.
		{
			name:      "fake",
			sourceDir: "/tmp/a",
			transport: "restic",
			logfile:   "/dev/null",
			wantError: true,
		},
	}

	for _, tt := range casetests {
		fakeExecute := NewFakeExecute()

		log := logger.New("")

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

		// Create a new restic object with our fakeExecute and a sinking outLogWriter.
		restic, err := NewResticTransport(cfg, fakeExecute, log, tt.dryRun)
		if tt.wantError && err != nil {
			continue
		}
		if err != nil {
			t.Fatalf("restic.NewResticTransport failed: %v", err)
		}

		if err := restic.Run(); err != nil {
			t.Fatalf("restic.Run failed: %v", err)
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
