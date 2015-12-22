// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"github.com/marcopaganini/netbackup/execute"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// This file contains all the shared infrastructure required for the transports
// tests. Individual transports tests go in their respective *_test.go files.

// FakeExecute is a fake implementation of execute.Execute that saves the executed
// command for later inspection by the caller.
type FakeExecute struct {
	cmd string
}

func NewFakeExecute() *FakeExecute {
	return &FakeExecute{}
}

func (f *FakeExecute) SetStdout(execute.CallbackFunc) {
}

func (f *FakeExecute) SetStderr(execute.CallbackFunc) {
}

func (f *FakeExecute) Cmd() string {
	return f.cmd
}

func (f *FakeExecute) Exec(a []string) error {
	f.cmd = strings.Join(a, " ")
	return nil
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

// Test absPaths
func TestAbsPaths(t *testing.T) {
	base := "/foo/bar"
	paths := []string{"aaa", "bbb", "ccc"}
	expected := []string{base + "/aaa", base + "/bbb", base + "/ccc"}

	for ix, v := range absPaths(base, paths) {
		if v != expected[ix] {
			t.Errorf("absPaths failed at index %d; expected %q, got %q", ix, expected[ix], v)
		}
	}
}
