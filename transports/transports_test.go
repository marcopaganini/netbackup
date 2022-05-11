// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015-2019 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"context"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/execute"
)

// This file contains all the shared infrastructure required for the transports
// tests. Individual transports tests go in their respective *_test.go files.

// FakeExecute is a fake implementation of execute.Execute that saves the executed
// commands for later inspection by the caller.
type FakeExecute struct {
	cmds []string
}

func NewFakeExecute() *FakeExecute {
	return &FakeExecute{}
}

func (f *FakeExecute) SetStdout(execute.CallbackFunc) {
}

func (f *FakeExecute) SetStderr(execute.CallbackFunc) {
}

func (f *FakeExecute) Cmds() []string {
	return f.cmds
}

func (f *FakeExecute) Exec(a []string) error {
	f.cmds = append(f.cmds, strings.Join(a, " "))
	return nil
}

// Test writeList
func TestWriteList(t *testing.T) {
	log := logger.New("")
	ctx := context.Background()
	logger.WithLogger(ctx, log)

	items := []string{"aa", "aa/01", "aa/02", "bb"}
	fname, err := writeList(ctx, "fakename", items)
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

// reMatch returns true if all all strings in a slice match regular expressions in
// another slice, 1:1.
func reMatch(re, s []string) (bool, error) {
	if len(re) != len(s) {
		return false, nil
	}
	for i, r := range re {
		matched, err := regexp.MatchString(r, s[i])
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}
