// common.go: Common routines for transports
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package transports

import (
	"fmt"
	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/runner"
	"io"
	"io/ioutil"
	"os"
)

// CommandRunner defines the interface used to run commands.
type CommandRunner interface {
	SetStdout(runner.CallbackFunc)
	SetStderr(runner.CallbackFunc)
	Exec([]string) error
}

// Transport represents all transports
type Transport struct {
	config *config.Config
	runner CommandRunner
	outLog io.Writer
	log    *logger.Logger
	dryRun bool
}

// writeList writes the desired list of exclusions/inclusions into a file, in a
// format suitable for this transport. The caller is responsible for deleting
// the file after use. Returns the name of the file and error.
func writeList(prefix string, patterns []string) (string, error) {
	var w *os.File
	var err error

	if w, err = ioutil.TempFile("/tmp", prefix); err != nil {
		return "", fmt.Errorf("Error creating pattern file for %s list: %v", prefix, err)
	}
	defer w.Close()
	for _, v := range patterns {
		fmt.Fprintln(w, v)
	}
	return w.Name(), nil
}
