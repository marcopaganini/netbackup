// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package config_test

import (
	"github.com/marcopaganini/netbackup/config"
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
	cstr := "name=foo\ntransport=transp\nsource_dir=src\ndest_dir=dst"
	r := strings.NewReader(cstr)

	cfg, err := config.ParseConfig(r)
	if err != nil {
		t.Fatal("ParseConfig failed:", err)
	}
	if cfg.Name != "foo" {
		t.Errorf("name should be foo; is %s", cfg.Name)
	}
	if cfg.Transport != "transp" {
		t.Errorf("name should be transp; is %s", cfg.Name)
	}
	if cfg.SourceDir != "src" {
		t.Errorf("source_dir should be src; is %s", cfg.SourceDir)
	}
	if cfg.DestDir != "dst" {
		t.Errorf("dest_dir should be dst; is %s", cfg.DestDir)
	}
}

// Test that invalid key generates an exception.
func TestParseConfigInvalidKey(t *testing.T) {
	cstr := "name=foo\ntransport=transp\ninvalidkey=foo"
	r := strings.NewReader(cstr)

	if _, err := config.ParseConfig(r); err == nil {
		t.Fatalf("ParseConfig succeeded with invalid key; want non-nil error:", err)
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
				s += v + "=dummy\n"
			}
		}
		r := strings.NewReader(s)
		if _, err := config.ParseConfig(r); err == nil {
			t.Fatalf("ParseConfig succeeded when key %q is missing; want non-nil error:", miss, err)
		}
	}
}

// Test that Exclude and Include produce lists of strings.
func TestParseConfigLists(t *testing.T) {
	cstr := "name=foo\ntransport=transp\nsource_dir=src\ndest_dir=dst\nexclude=aa bb cc\ninclude=dd ee ff"
	r := strings.NewReader(cstr)

	cfg, err := config.ParseConfig(r)
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
