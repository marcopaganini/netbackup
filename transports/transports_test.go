// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"github.com/marcopaganini/netbackup/runner"
	"strings"
)

// This file contains all the shared infrastructure required for the transports
// tests. Individual transports tests go in their respective *_test.go files.

// FakeRunner is a fake implementation of runner.Runner that saves the executed
// command for later inspection by the caller.
type FakeRunner struct {
	cmd string
}

func NewFakeRunner() *FakeRunner {
	return &FakeRunner{}
}

func (f *FakeRunner) SetStdout(runner.CallbackFunc) {
}

func (f *FakeRunner) SetStderr(runner.CallbackFunc) {
}

func (f *FakeRunner) Cmd() string {
	return f.cmd
}

func (f *FakeRunner) Exec(a []string) error {
	f.cmd = strings.Join(a, " ")
	return nil
}
