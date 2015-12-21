// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"github.com/marcopaganini/netbackup/config"
	"os"
	"regexp"
	"testing"
	"testing/iotest"
)

const (
	rdiffBackupTestCmd = "rdiff-backup --verbosity=5 --terminal-verbosity=5 --preserve-numerical-ids --exclude-sockets --exclude-other-filesystems --force"
)

// Common rdiff-backup tests: Initialize an rdiff-backup instance with the
// config passed and test fail if the generated command doesn't match the
// expected command line regexp. The value of dryRun is passed to the
// initializon of rdiff-backup. If must_error is true, the call to
// NewRdiffBackupTransport *must* fail, or the test will fail.
func rdiffBackupTest(t *testing.T, cfg *config.Config, expect string, dryRun bool, mustError bool) {
	fakeExecute := NewFakeExecute()

	// Create a new rdiff-backup object with our fakeExecute and a sinking outLogWriter.
	rdiffBackup, err := NewRdiffBackupTransport(cfg, fakeExecute, iotest.TruncateWriter(os.Stderr, 0), 0, dryRun)
	if err != nil {
		if mustError {
			return
		}
		t.Fatalf("NewRdiffBackupTransport failed: %v", err)
	}
	if mustError {
		t.Fatalf("NewRdiffBackupTransport should return have failed, but got nil error")
	}
	if err := rdiffBackup.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	matched, err := regexp.MatchString(expect, fakeExecute.Cmd())
	if err != nil {
		t.Fatalf("error during regexp match: %v", err)
	}
	if !matched {
		t.Fatalf("name should match %s; is %s", expect, fakeExecute.Cmd())
	}
}

// Dry run: No command should be executed
func TestRdiffBackupDryRun(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Transport: "rdiff-backup",
		Logfile:   "/dev/null",
	}
	rdiffBackupTest(t, cfg, "", true, false)
}

// Same machine (local copy)
func TestRdiffBackupSameMachine(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Transport: "rdiff-backup",
		Logfile:   "/dev/null",
	}
	rdiffBackupTest(t, cfg, rdiffBackupTestCmd+" /tmp/a /tmp/b", false, false)
}

// Local source, remote destination
func TestRdiffBackupLocalSourceRemoteDest(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		DestHost:  "desthost",
		Transport: "rdiff-backup",
		Logfile:   "/dev/null",
	}
	rdiffBackupTest(t, cfg, rdiffBackupTestCmd+" /tmp/a desthost::/tmp/b", false, false)
}

// Remote source, local destination (unusual)
func TestRdiffBackupRemoteSourceLocalDest(t *testing.T) {
	cfg := &config.Config{
		Name:       "fake",
		SourceHost: "srchost",
		SourceDir:  "/tmp/a",
		DestDir:    "/tmp/b",
		Transport:  "rdiff-backup",
		Logfile:    "/dev/null",
	}
	rdiffBackupTest(t, cfg, rdiffBackupTestCmd+" srchost::/tmp/a /tmp/b", false, false)
}

// Remote source, Remote destination (server side copy) not supported under
// rdiff-backup and should return an error.
func TestRdiffBackupRemoteSourceRemoteDest(t *testing.T) {
	cfg := &config.Config{
		Name:       "fake",
		SourceHost: "srchost",
		SourceDir:  "/tmp/a",
		DestHost:   "desthost",
		DestDir:    "/tmp/b",
		Transport:  "rdiff-backup",
		Logfile:    "/dev/null",
	}
	rdiffBackupTest(t, cfg, "", false, true)
}

// Exclude list only
func TestRdiffBackupExcludeListOnly(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Exclude:   []string{"x/foo", "x/bar"},
		Transport: "rdiff-backup",
		Logfile:   "/dev/null",
	}
	rdiffBackupTest(t, cfg, rdiffBackupTestCmd+" --exclude-globbing-filelist=[^ ]+ /tmp/a /tmp/b", false, false)
}

// Include list only
func TestRdiffBackupIncludeListOnly(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Include:   []string{"x/foo", "x/bar"},
		Transport: "rdiff-backup",
		Logfile:   "/dev/null",
	}
	rdiffBackupTest(t, cfg, rdiffBackupTestCmd+" --include-globbing-filelist=[^ ]+ /tmp/a /tmp/b", false, false)
}

// Include & Exclude lists
func TestRdiffBackupIncludeAndExclude(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Exclude:   []string{"x/foo", "x/bar"},
		Include:   []string{"x/foo", "x/bar"},
		Transport: "rdiff-backup",
		Logfile:   "/dev/null",
	}
	rdiffBackupTest(t, cfg, rdiffBackupTestCmd+" --exclude-globbing-filelist=[^ ]+ --include-globbing-filelist=[^ ]+ /tmp/a /tmp/b", false, false)
}

// Test that an empty source dir results in an error
func TestRdiffBackupEmptySourceDir(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		DestDir:   "/tmp/b",
		Transport: "rdiff-backup",
		Logfile:   "/dev/null",
	}
	rdiffBackupTest(t, cfg, "", false, true)
}

// Test that an empty destination dir results in an error
func TestRdiffBackupEmptyDestDir(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		Transport: "rdiff-backup",
		Logfile:   "/dev/null",
	}
	rdiffBackupTest(t, cfg, "", false, true)
}
