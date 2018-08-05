// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015-2018 by Marco Paganini <paganini AT paganini DOT net>

package config

import (
	"strings"
	"testing"
)

// compare two arrays. Return true if they're the same, false otherwise.
func arrayEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for ix := 0; ix < len(a); ix++ {
		if a[ix] != b[ix] {
			return false
		}
	}
	return true
}

// Test minimal configuration.
func TestParseConfigMinimal(t *testing.T) {
	cstr := "name=\"foo\"\ntransport=\"transp\"\nsource_dir=\"/src\"\ndest_dir=\"/dst\""
	r := strings.NewReader(cstr)

	cfg, err := ParseConfig(r)
	if err != nil {
		t.Fatal("ParseConfig failed:", err)
	}
	if cfg.Name != "foo" {
		t.Errorf("name should be foo; is %s", cfg.Name)
	}
	if cfg.Transport != "transp" {
		t.Errorf("name should be transp; is %s", cfg.Name)
	}
	if cfg.SourceDir != "/src" {
		t.Errorf("source_dir should be /src; is %s", cfg.SourceDir)
	}
	if cfg.DestDir != "/dst" {
		t.Errorf("dest_dir should be /dst; is %s", cfg.DestDir)
	}
}

// Test that invalid key generates an exception.
func TestParseConfigInvalidKey(t *testing.T) {
	cstr := "name=\"foo\"\ntransport=\"transp\"\ninvalidkey=\"foo\""
	r := strings.NewReader(cstr)

	if _, err := ParseConfig(r); err == nil {
		t.Fatalf("ParseConfig succeeded with invalid key; want non-nil error: %v", err)
	}
}

// Test that lack of mandatory fields generates an error.
func TestParseConfigMandatoryMissing(t *testing.T) {
	// List of mandatory fields. Make sure ONLY mandatory keys are listed here.
	mandatory := []string{"name", "transport", "source_dir", "dest_dir"}

	// Generate one config for each of the mandatory fields missing
	for _, miss := range mandatory {
		s := ""
		// Generate config
		for _, v := range mandatory {
			if v != miss {
				s += v + "=\"dummy\"\n"
			}
		}
		r := strings.NewReader(s)
		if _, err := ParseConfig(r); err == nil {
			t.Fatalf("ParseConfig succeeded when key %q is missing; want non-nil error", miss)
		}
	}
}

// Make sure that improper combinations of destinations produce an error.
func TestDestOptions(t *testing.T) {
	baseConfig := "name=\"foo\"\ntransport=\"transp\"\nsource_dir=\"/src\"\n"

	// dest_dir and dest_dev should result in error.
	r := strings.NewReader(baseConfig + "dest_dir=\"/dst\"\ndest_dev=\"/dev/foo\"")
	if _, err := ParseConfig(r); err == nil {
		t.Fatalf("ParseConfig succeeded when dest_dir and dest_dev are set; want non-nil error")
	}

	// dest_dev and dest_host should result in error.
	r = strings.NewReader(baseConfig + "dest_dev=\"/dev/foo\"\ndest_host=\"foohost\"")
	if _, err := ParseConfig(r); err == nil {
		t.Fatalf("ParseConfig succeeded when key dest_dev and dest_host are set; want non-nil error")
	}

	// dest_dev and dest_luks_dev should result in error.
	r = strings.NewReader(baseConfig + "dest_dev=\"/dev/foo\"\ndest_luks_dev=\"/luksdev\"\nluks_key_file=\"foo\"")
	if _, err := ParseConfig(r); err == nil {
		t.Fatalf("ParseConfig succeeded when key dest_dev and luks_dest_dev are set; want non-nil error")
	}

	// dest_luks_dev without a key file should result in error.
	r = strings.NewReader(baseConfig + "dest_luks_dev=\"/luksdev\"\nluks_key_file=\"foo\"")
	if _, err := ParseConfig(r); err == nil {
		t.Fatalf("ParseConfig succeeded when key luks_dest_dev is set without a luks_kefile; want non-nil error")
	}

	// filesystem_cleanup without a filesystem destination should result in error.
	r = strings.NewReader(baseConfig + "dest_dir=\"/dst\"\nfs_cleanup=\"yes\"")
	if _, err := ParseConfig(r); err == nil {
		t.Fatalf("ParseConfig succeeded when key luks_dest_dev is set without a luks_kefile; want non-nil error")
	}
}

// Test source_is_mountpoint options.
func TestSourceIsMountpoint(t *testing.T) {
	baseConfig := "name=\"foo\"\ntransport=\"transp\"\nsource_dir=\"/src\"\ndest_dir=\"/dst\"\nsource_is_mountpoint=true\n"

	// Make sure source_is_mountpoint set to true doesn't cause an error
	r := strings.NewReader(baseConfig)
	if _, err := ParseConfig(r); err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	// Make sure source_is_mountpoint set with source_host set results in error.
	r = strings.NewReader(baseConfig + "source_host=\"meh\"\n")
	if _, err := ParseConfig(r); err == nil {
		t.Errorf("ParseConfig succeeded when source_is_mountpoint and source_host are set; want non-nil error")
	}
}

// Make sure that improper Logging combinations produce errors and that
// defaults are assigned to LogDir if it's empty.
func TestLoggingOptions(t *testing.T) {
	baseConfig := "name=\"foo\"\ntransport=\"transp\"\nsource_dir=\"/src\"\ndest_dir=\"/dst\"\n"
	logDir := "/logdir"
	logFile := "/logfile"

	// LogDir and no Logfile
	r := strings.NewReader(baseConfig + "log_dir=\"" + logDir + "\"")
	cfg, err := ParseConfig(r)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}
	if cfg.LogDir != logDir {
		t.Errorf("log_dir should be %s; is %s", logDir, cfg.LogDir)
	}
	if cfg.Logfile != "" {
		t.Errorf("log_dir should be empty; is %s", cfg.Logfile)
	}

	// Logfile and no LogDir
	r = strings.NewReader(baseConfig + "log_file=\"" + logFile + "\"")
	cfg, err = ParseConfig(r)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}
	if cfg.Logfile != logFile {
		t.Errorf("log_file should be %s; is %s", logFile, cfg.Logfile)
	}
	if cfg.LogDir != "" {
		t.Errorf("log_dir should be empty; is %s", cfg.LogDir)
	}

	// Blank Logfile & Blank LogDir == default LogDir
	r = strings.NewReader(baseConfig)
	cfg, err = ParseConfig(r)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}
	if cfg.LogDir != defaultLogDir {
		t.Errorf("log_dir should be %s; is %s", defaultLogDir, cfg.LogDir)
	}
	if cfg.Logfile != "" {
		t.Errorf("log_file should be empty; is %s", cfg.Logfile)
	}

	// Logfile and LogDir should result in error
	r = strings.NewReader(baseConfig + "log_file=\"" + logFile + "\"\nlog_dir=\"" + logDir + "\"")
	if _, err := ParseConfig(r); err == nil {
		t.Errorf("ParseConfig succeeded when keys log_dir and log_file are set; want non-nil error")
	}
}

// Test that relative paths for source or destination dir result in error.
// if SourceHost and DestHost are not set (local backup), respectively.
func TestRelativePaths(t *testing.T) {
	// Relative source_dir, local backup (FAIL)
	cstr := "name=\"foo\"\nsource_dir=\"a\"\ndest_dir=\"/b\"\ntransport=\"transp\""
	r := strings.NewReader(cstr)
	if _, err := ParseConfig(r); err == nil {
		t.Fatalf("ParseConfig succeeded when source_dir is a relative path; want non-nil error: %v", err)
	}

	// Relative dest_dir, local backup (FAIL)
	cstr = "name=\"foo\"\nsource_dir=\"/a\"\ndest_dir=\"b\"\ntransport=\"transp\""
	r = strings.NewReader(cstr)
	if _, err := ParseConfig(r); err == nil {
		t.Fatalf("ParseConfig succeeded when dest_dir is a relative path; want non-nil error: %v", err)
	}

	// Relative source_dir, sourc_host set (OK)
	cstr = "name=\"foo\"\nsource_dir=\"a\"\nsource_host=\"foo\"\ndest_dir=\"/b\"\ntransport=\"transp\""
	r = strings.NewReader(cstr)
	if _, err := ParseConfig(r); err != nil {
		t.Fatalf("ParseConfig failed when source_dir is a relative path and source_host is set: %v", err)
	}

	// Relative dest_dir, local backup
	cstr = "name=\"foo\"\nsource_dir=\"/a\"\ndest_dir=\"b\"\ndest_host=\"foo\"\ntransport=\"transp\""
	r = strings.NewReader(cstr)
	if _, err := ParseConfig(r); err != nil {
		t.Fatalf("ParseConfig failed when dest_dir is a relative path and dest_host is set: %v", err)
	}
}

// Test that Exclude and Include produce lists of strings.
func TestParseConfigLists(t *testing.T) {
	cstr := "name=\"foo\"\ntransport=\"transp\"\nsource_dir=\"/src\"\ndest_dir=\"/dst\"\nexclude=[\"aa\", \"bb\", \"cc\"]\ninclude=[\"dd\", \"ee\", \"ff\"]"
	r := strings.NewReader(cstr)

	cfg, err := ParseConfig(r)
	if err != nil {
		t.Fatal("ParseConfig failed:", err)
	}

	expected := []string{"aa", "bb", "cc"}
	if !arrayEqual(cfg.Exclude, expected) {
		t.Errorf("Exclude should be %s, is %s", expected, cfg.Name)
	}

	expected = []string{"dd", "ee", "ff"}
	if !arrayEqual(cfg.Include, expected) {
		t.Errorf("Include should be %s, is %s", expected, cfg.Name)
	}
}
