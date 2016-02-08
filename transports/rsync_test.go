// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"regexp"
	"testing"
)

const (
	rsyncTestCmd = "rsync -av --delete --numeric-ids"
)

// Common rsync tests: Initialize an rsync instance with the config passed
// and test fail if the generated command doesn't match the expected command
// line regexp. The value of dryRun is passed to the initializon of rsync. If
// must_error is true, the call to NewRsyncTransport *must* fail, or the test
// will fail.
func rsyncTest(t *testing.T, cfg *config.Config, expect string, dryRun bool, mustError bool) {
	fakeExecute := NewFakeExecute()

	log := logger.New("")

	// Create a new rsync object with our fakeExecute and a sinking outLogWriter.
	rsync, err := NewRsyncTransport(cfg, fakeExecute, log, dryRun)
	if err != nil {
		if mustError {
			return
		}
		t.Fatalf("rsync.NewRsyncTransport failed: %v", err)
	}
	if mustError {
		t.Fatalf("rsync.NewRsyncTransport should return have failed, but got nil error")
	}
	if err := rsync.Run(); err != nil {
		t.Fatalf("rsync.Run failed: %v", err)
	}
	matched, err := regexp.MatchString(expect, fakeExecute.Cmd())
	if err != nil {
		t.Fatalf("error during regexp match: %v", err)
	}
	if !matched {
		t.Fatalf("name should match %s; is %s", expect, fakeExecute.Cmd())
	}
}

// Dry run: No command should be executed.
func TestRsyncDryRun(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Transport: "rsync",
		Logfile:   "/dev/null",
	}
	rsyncTest(t, cfg, "", true, false)
}

// Same machine (local copy).
func TestRsyncSameMachine(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Transport: "rsync",
		Logfile:   "/dev/null",
	}
	rsyncTest(t, cfg, rsyncTestCmd+" /tmp/a/ /tmp/b", false, false)
}

// Local source, remote destination.
func TestRsyncLocalSourceRemoteDest(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		DestHost:  "desthost",
		Transport: "rsync",
		Logfile:   "/dev/null",
	}
	rsyncTest(t, cfg, rsyncTestCmd+" /tmp/a/ desthost:/tmp/b", false, false)
}

// Remote source, local destination.
func TestRsyncRemoteSourceLocalDest(t *testing.T) {
	cfg := &config.Config{
		Name:       "fake",
		SourceHost: "srchost",
		SourceDir:  "/tmp/a",
		DestDir:    "/tmp/b",
		Transport:  "rsync",
		Logfile:    "/dev/null",
	}
	rsyncTest(t, cfg, rsyncTestCmd+" srchost:/tmp/a/ /tmp/b", false, false)
}

// Remote source, Remote destination (server side copy)not supported by rsync.
func TestRsyncRemoteSourceRemoteDest(t *testing.T) {
	cfg := &config.Config{
		Name:       "fake",
		SourceHost: "srchost",
		SourceDir:  "/tmp/a",
		DestHost:   "desthost",
		DestDir:    "/tmp/b",
		Transport:  "rsync",
		Logfile:    "/dev/null",
	}
	rsyncTest(t, cfg, "", false, true)
}

// Sources ending in a slash should not have another slash added.
func TestRsyncDoubleSlash(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/",
		DestDir:   "/tmp/b",
		Transport: "rsync",
		Logfile:   "/dev/null",
	}
	rsyncTest(t, cfg, rsyncTestCmd+" / /tmp/b", false, false)
}

// Exclude list only.
func TestRsyncExcludeListOnly(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Exclude:   []string{"x/foo", "x/bar"},
		Transport: "rsync",
		Logfile:   "/dev/null",
	}
	rsyncTest(t, cfg, rsyncTestCmd+" --exclude-from=[^ ]+ --delete-excluded /tmp/a/ /tmp/b", false, false)
}

// Include list only.
func TestRsyncIncludeListOnly(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Include:   []string{"x/foo", "x/bar"},
		Transport: "rsync",
		Logfile:   "/dev/null",
	}
	rsyncTest(t, cfg, rsyncTestCmd+" --include-from=[^ ]+ /tmp/a/ /tmp/b", false, false)
}

// Include & Exclude lists.
func TestRsyncIncludeAndExclude(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Exclude:   []string{"x/foo", "x/bar"},
		Include:   []string{"x/foo", "x/bar"},
		Transport: "rsync",
		Logfile:   "/dev/null",
	}
	rsyncTest(t, cfg, rsyncTestCmd+" --exclude-from=[^ ]+ --delete-excluded --include-from=[^ ]+ /tmp/a/ /tmp/b", false, false)
}

// Test that an empty source dir results in error.
func TestRsyncEmptySourceDir(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		DestDir:   "/tmp/b",
		Transport: "rsync",
		Logfile:   "/dev/null",
	}
	rsyncTest(t, cfg, "", false, true)
}

// Test that an empty destination dir results in error.
func TestRsyncEmptyDestDir(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		Transport: "rsync",
		Logfile:   "/dev/null",
	}
	rsyncTest(t, cfg, "", false, true)
}
