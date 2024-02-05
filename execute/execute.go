// This file is part of netbackup, a frontend to simplify periodic backups.
// For further information, check https://github.com/marcopaganini/netbackup
//
// (C) 2015-2024 by Marco Paganini <paganini AT paganini DOT net>

package execute

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/marcopaganini/logger"
)

// CallbackFunc represents callback functions functions for stdout/stderr output
type CallbackFunc func(string) error

// Executor defines the interface used to run commands.
type Executor interface {
	SetStdout(CallbackFunc)
	SetStderr(CallbackFunc)
	Exec([]string) error
}

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

// hmsNow returns the current time in HMS format (hour minute second)
func hmsNow() string {
	return time.Now().Format("15:04:05")
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

// matchSlice returns true if the string s matches any substring within
// the passed slice, false otherwise.
func matchSlice(slice []string, s string) bool {
	for _, v := range slice {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
}

// ExitCode fetches the numeric return code from the return of RunCommand.
// There's no portable way of retrieving the exit code. This function returns
// 255 if there is an error in the code and we are in a platform that does not
// have syscall.WaitStatus.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	retcode := 255
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			retcode = status.ExitStatus()
		}
	}
	return retcode
}

// WithShell receives a string command and returns an slice ready to be passed
// to Run or RunCommand with the current shell prepended to it.  The function
// works as a helper to run strings commands using the shell with Run or
// RunCommand.
func WithShell(cmd string) []string {
	// Run using shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return []string{shell, "-c", "--", cmd}
}

// Run executes the given command using the prefix. Output is logged using the
// supplied logger object. This is a convenience function around RunCommand,
// since most command invocations don't need the extra functionality supplied
// by that function.
func Run(ctx context.Context, prefix string, cmd []string) error {
	return RunCommand(ctx, prefix, cmd, nil, nil, nil)
}

// RunCommand executes the given command using the supplied Execute object. The
// method logs the output of the program (stdout/err) using the logger object,
// with a verbosity level of 3. Every output line is prefixed by the current
// HMS. If the Execute object is nil, a new one will be created. outFilter and
// errFilter contain optional slices of substrings which, if matched, will
// cause the entire line to be excluded from the output.
func RunCommand(ctx context.Context, prefix string, cmd []string, ex Executor, outFilter []string, errFilter []string) error {
	log := logger.LoggerValue(ctx)

	log.Verbosef(2, "%s Start: %s\n", prefix, time.Now().Format(time.Stamp))
	log.Verbosef(1, "%s Command: %q\n", prefix, strings.Join(cmd, " "))

	// Create a new execute object, if current is nil
	e := ex
	if e == nil {
		e = New()
	}

	// Filter functions: These functions will copy stderr and stdout to
	// the log, omitting lines that match our filters.
	errFilterFunc := func(buf string) error {
		if errFilter == nil || !matchSlice(errFilter, buf) {
			log.Verbosef(3, "%s (err): %s\n", hmsNow(), buf)
			return nil
		}
		return nil
	}
	outFilterFunc := func(buf string) error {
		if outFilter == nil || !matchSlice(outFilter, buf) {
			log.Verbosef(3, "%s (out): %s\n", hmsNow(), buf)
			return nil
		}
		return nil
	}

	// All streams copied to output log with date as a prefix.
	e.SetStderr(errFilterFunc)
	e.SetStdout(outFilterFunc)

	err := e.Exec(cmd)
	log.Verbosef(2, "%s Finish: %s\n", prefix, time.Now().Format(time.Stamp))
	if err != nil {
		log.Verbosef(1, "%s returned: %v\n", prefix, err)
		return err
	}
	log.Verbosef(1, "%s: returned: OK\n", prefix)
	return nil
}
