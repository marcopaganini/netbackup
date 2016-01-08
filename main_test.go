// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
)

// Return a Backup object with fake calls
func fakeBackup() *Backup {
	log = logger.New("")
	fakeConfig := &config.Config{}

	fakeBackup := &Backup{
		log:     log,
		config:  fakeConfig,
		verbose: 0,
		dryRun:  false}
	return fakeBackup
}

// Test createOutputLog
func TestCreateOutputLog(t *testing.T) {
	w, err := ioutil.TempFile("/tmp/", "test")
	if err != nil {
		t.Fatalf("Tempdir failed: %v", err)
	}
	testFname := w.Name()
	w.Close()

	// Test specific file under /tmp. File must exist at the end.
	w, err = logOpen(testFname)
	if err != nil {
		t.Fatalf("CreateOutputLog failed: %v", err)
	}
	w.Close()
	if _, err := os.Stat(testFname); err != nil {
		t.Errorf("should be able to open %s; got %v", testFname, err)
	}
	os.Remove(testFname)

	// Test that intermediate directories are created
	w, err = logOpen("/tmp/a/b/c/log")
	if err != nil {
		t.Fatalf("CreateOutputLog failed: %v", err)
	}
	w.Close()

	// File must match the expected name and exist
	expected := "/tmp/a/b/c/log"
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Errorf("%s not created", expected)
	}
	os.RemoveAll("/tmp/a")
}
