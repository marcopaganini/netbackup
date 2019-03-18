// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015-2019 by Marco Paganini <paganini AT paganini DOT net>

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
)

// Return a Backup object with fake calls
func fakeBackup() *Backup {
	log = logger.New("")
	fakeConfig := &config.Config{}

	fakeBackup := &Backup{
		log:    log,
		config: fakeConfig,
		dryRun: false}
	return fakeBackup
}

// Test logOpen
func TestLogOpen(t *testing.T) {
	w, err := ioutil.TempFile("/tmp/", "test")
	if err != nil {
		t.Fatalf("TempFile failed: %v", err)
	}
	testFname := w.Name()
	w.Close()

	// Test specific file under /tmp. File must exist at the end.
	w, err = logOpen(testFname)
	if err != nil {
		t.Fatalf("logOpen failed: %v", err)
	}
	w.Close()
	if _, err := os.Stat(testFname); err != nil {
		t.Errorf("should be able to open %s; got %v", testFname, err)
	}
	os.Remove(testFname)

	// Test that intermediate directories are created
	basedir, err := ioutil.TempDir("/tmp", "netbackup_test")
	if err != nil {
		t.Errorf("error creating temporary dir: %v", err)
	}
	logpath := "a/b/c/log"

	w, err = logOpen(filepath.Join(basedir, logpath))
	if err != nil {
		t.Fatalf("logOpen failed: %v", err)
	}
	w.Close()

	// File must match the expected name and exist
	expected := filepath.Join(basedir, logpath)
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Errorf("%s not created", expected)
	}
	os.RemoveAll(basedir)
}
