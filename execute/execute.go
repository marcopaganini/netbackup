// Runs an external binary and redirect stdout/stderr to an io.Writer
// using user-supplied functions to filter the output content.

// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package execute

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

// CallbackFunc represents callback functions functions for stdout/stderr output
type CallbackFunc func(string) error

// Execute defines a struct to easily run external programs and
// capture their stdout and stderr.
type Execute struct {
	outWrite CallbackFunc
	errWrite CallbackFunc
}

// New returns a new Execute object
func New() *Execute {
	return &Execute{
		outWrite: nil,
		errWrite: nil,
	}
}

// SetStdout sets the stdout processing function
func (e *Execute) SetStdout(f CallbackFunc) {
	e.outWrite = f
}

// SetStderr sets the stderr processing function
func (e *Execute) SetStderr(f CallbackFunc) {
	e.errWrite = f
}

// Exec runs a program specified in the slice cmd. The first element of the
// slice is used as the executable name, and the rest as the arguments.  The
// standard output and standard error of the executed program will be sent
// line-by-line to outWrite() and errWrite() respectively. These (user
// supplied) functions may decide to write to a file, file-descriptor or ignore
// each of the lines in the output. Returns the error value from exec.Wait()
func (e *Execute) Exec(cmd []string) error {
	run := exec.Command(cmd[0], cmd[1:]...)

	// Grab stdout & stderr
	stdout, err := run.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := run.StderrPipe()
	if err != nil {
		return err
	}

	// Start command
	if err := run.Start(); err != nil {
		return err
	}

	// Channels
	outchan := make(chan error, 1)
	errchan := make(chan error, 1)

	go stream(stdout, e.outWrite, outchan)
	go stream(stderr, e.errWrite, errchan)

	// Wait until goroutines exhaust stdout and stderr
	// Capture error from streamig goroutine (if any)
	err = <-outchan
	if err != nil {
		return fmt.Errorf("Error reading program's stdout: %v", err)
	}
	err = <-errchan
	if err != nil {
		return fmt.Errorf("Error reading program's stderr: %v", err)
	}

	return run.Wait()
}

// stream reads lines from an io.ReadCloser and calls outFunc() with each of
// the lines as a string. If outFunc() returns an error, control immediately
// returns to the parent.
func stream(r io.ReadCloser, outFunc CallbackFunc, c chan error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		if err := outFunc(s.Text()); err != nil {
			c <- err
			return
		}
	}
	c <- nil
}
