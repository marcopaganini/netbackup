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
)

// Test createOutputLog
func TestCreateOutputLog(t *testing.T) {
	w, err := ioutil.TempFile("/tmp/", "test")
	if err != nil {
		t.Fatalf("Tempdir failed: %v", err)
	}
	testFname := w.Name()
	w.Close()

	// Test specific file under /tmp. File must exist.
	w, fname, err := createOutputLog(testFname, "", "")
	if err != nil {
		t.Fatalf("CreateOutputLog failed: %v", err)
	}
	w.Close()

	// Returned filename must match the passed one.
	if fname != testFname {
		t.Errorf("fname should be %s; is %s", testFname, fname)
	}
	os.Remove(fname)

	// Test automatic file generation. A file named
	// $dir/backup_name/backup_name-yyyy-mm-dd.log
	// should be created.
	cname := "dummy"
	w, fname, err = createOutputLog("", "/tmp", cname)
	if err != nil {
		t.Fatalf("CreateOutputLog failed: %v", err)
	}
	w.Close()

	// File must match the expected name and exist
	expected := fmt.Sprintf("/tmp/%s/%s-%s.log", cname, cname, time.Now().Format("2006-01-02"))
	if fname != expected {
		t.Errorf("fname should be %s; is %s", expected, fname)
	}
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		t.Errorf("%s not created", fname)
	}
	os.RemoveAll(filepath.Join("/tmp", cname))
}
