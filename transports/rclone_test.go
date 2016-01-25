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

// Common rclone tests: Initialize an rclone instance with the config passed
// and test fail if the generated command doesn't match the expected command
// line regexp. The value of dryRun is passed to the initializon of rclone. If
// must_error is true, the call to NewRcloneTransport *must* fail, or the test
// will fail.
func rcloneTest(t *testing.T, cfg *config.Config, expect string, dryRun bool, mustError bool) {
	fakeExecute := NewFakeExecute()

	log := logger.New("")

	// Create a new rclone object with our fakeExecute and a sinking outLogWriter.
	rclone, err := NewRcloneTransport(cfg, fakeExecute, log, dryRun)
	if err != nil {
		if mustError {
			return
		}
		t.Fatalf("rclone.NewRcloneTransport failed: %v", err)
	}
	if mustError {
		t.Fatalf("rclone.NewRcloneTransport should return have failed, but got nil error")
	}
	if err := rclone.Run(); err != nil {
		t.Fatalf("rclone.Run failed: %v", err)
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
func TestRcloneDryRun(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "", true, false)
}

// Same machine (local copy)
func TestRcloneSameMachine(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "rclone sync -v /tmp/a /tmp/b", false, false)
}

// Local source, remote destination
func TestRcloneLocalSourceRemoteDest(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		DestHost:  "desthost",
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "rclone sync -v /tmp/a desthost:/tmp/b", false, false)
}

// Remote source, local destination (unusual)
func TestRcloneRemoteSourceLocalDest(t *testing.T) {
	cfg := &config.Config{
		Name:       "fake",
		SourceHost: "srchost",
		SourceDir:  "/tmp/a",
		DestDir:    "/tmp/b",
		Transport:  "rclone",
		Logfile:    "/dev/null",
	}
	rcloneTest(t, cfg, "rclone sync -v srchost:/tmp/a /tmp/b", false, false)
}

// Remote source, Remote destination (server side copy)
func TestRcloneRemoteSourceRemoteDest(t *testing.T) {
	cfg := &config.Config{
		Name:       "fake",
		SourceHost: "srchost",
		SourceDir:  "/tmp/a",
		DestHost:   "desthost",
		DestDir:    "/tmp/b",
		Transport:  "rclone",
		Logfile:    "/dev/null",
	}
	rcloneTest(t, cfg, "rclone sync -v srchost:/tmp/a desthost:/tmp/b", false, false)
}

// Exclude list only
func TestRcloneExcludeListOnly(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Exclude:   []string{"x/foo", "x/bar"},
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "rclone sync -v --exclude-from=[^ ]+ /tmp/a /tmp/b", false, false)
}

// Include list only
func TestRcloneIncludeListOnly(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Include:   []string{"x/foo", "x/bar"},
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "rclone sync -v --include-from=[^ ]+ /tmp/a /tmp/b", false, false)
}

// Include & Exclude lists
func TestRcloneIncludeAndExclude(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Exclude:   []string{"x/foo", "x/bar"},
		Include:   []string{"x/foo", "x/bar"},
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "rclone sync -v --exclude-from=[^ ]+ --include-from=[^ ]+ /tmp/a /tmp/b", false, false)
}

// Test that an empty source dir results in an error
func TestRcloneEmptySourceDir(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		DestDir:   "/tmp/b",
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "", false, true)
}

// Test that an empty destination dir results in an error
func TestRcloneEmptyDestDir(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "", false, true)
}
