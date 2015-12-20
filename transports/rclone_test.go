// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"github.com/marcopaganini/netbackup/config"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"
	"testing/iotest"
)

// Common rclone tests: Initialize an rclone instance with the config passed
// and test fail if the generated command doesn't match the expected command
// line regexp. The value of dryRun is passed to the initializon of rclone. If
// must_error is true, the call to NewRcloneTransport *must* fail, or the test
// will fail.
func rcloneTest(t *testing.T, cfg *config.Config, expect string, dryRun bool, mustError bool) {
	fakeExecute := NewFakeExecute()

	// Create a new rclone object with our fakeExecute and a sinking outLogWriter.
	rclone, err := NewRcloneTransport(cfg, fakeExecute, iotest.TruncateWriter(os.Stderr, 0), 0, dryRun)
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
func TestDryRun(t *testing.T) {
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
func TestSameMachine(t *testing.T) {
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
func TestLocalSourceRemoteDest(t *testing.T) {
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
func TestRemoteSourceLocalDest(t *testing.T) {
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
func TestRemoteSourceRemoteDest(t *testing.T) {
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
func TestExcludeListOnly(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Exclude:   []string{"x/foo", "x/bar"},
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "rclone sync -v --exclude=[^ ]+ /tmp/a /tmp/b", false, false)
}

// Include list only
func TestIncludeListOnly(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Include:   []string{"x/foo", "x/bar"},
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "rclone sync -v --include=[^ ]+ /tmp/a /tmp/b", false, false)
}

// Include & Exclude lists
func TestIncludeAndExclude(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		DestDir:   "/tmp/b",
		Exclude:   []string{"x/foo", "x/bar"},
		Include:   []string{"x/foo", "x/bar"},
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "rclone sync -v --exclude=[^ ]+ --include=[^ ]+ /tmp/a /tmp/b", false, false)
}

// Test that an empty source dir results in an error
func TestEmptySourceDir(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		DestDir:   "/tmp/b",
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "", false, true)
}

// Test that an empty destination dir results in an error
func TestEmptyDestDir(t *testing.T) {
	cfg := &config.Config{
		Name:      "fake",
		SourceDir: "/tmp/a",
		Transport: "rclone",
		Logfile:   "/dev/null",
	}
	rcloneTest(t, cfg, "", false, true)
}

// Test writeList
func TestWriteList(t *testing.T) {
	items := []string{"aa", "aa/01", "aa/02", "bb"}
	fname, err := writeList("fakename", items)
	if err != nil {
		t.Fatalf("writeList failed: %v", err)
	}
	contents, err := ioutil.ReadFile(fname)
	os.Remove(fname)
	if err != nil {
		t.Fatalf("Unable to read list file %q: %v", fname, err)
	}
	expected := strings.Join(items, "\n") + "\n"
	if string(contents) != expected {
		t.Fatalf("generated list file contents should match\n[%s]\n\nbut is\n\n[%s]", expected, string(contents))
	}
}
