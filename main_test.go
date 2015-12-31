// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	b := fakeBackup()
	b.config.Logfile = testFname

	err = b.createOutputLog("")
	if err != nil {
		t.Fatalf("CreateOutputLog failed: %v", err)
	}
	b.outLog.Close()
	if _, err := os.Stat(testFname); err != nil {
		t.Errorf("should be able to open %s; got %v", testFname, err)
	}
	os.Remove(testFname)

	// Test automatic file generation. A file named
	// $dir/backup_name/backup_name-yyyy-mm-dd.log
	// should be created.
	b = fakeBackup()
	b.config.Name = "dummy"
	err = b.createOutputLog("/tmp")
	if err != nil {
		t.Fatalf("CreateOutputLog failed: %v", err)
	}
	b.outLog.Close()

	// File must match the expected name and exist
	expected := fmt.Sprintf("/tmp/%s/%s-%s.log", b.config.Name, b.config.Name, time.Now().Format("2006-01-02"))
	if b.outLog.Name() != expected {
		t.Errorf("fname should be %s; is %s", expected, b.outLog.Name())
	}
	if _, err := os.Stat(b.outLog.Name()); os.IsNotExist(err) {
		t.Errorf("%s not created", b.outLog.Name())
	}
	os.RemoveAll(filepath.Join("/tmp", b.config.Name))
}
