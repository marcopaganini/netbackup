package transports

// common.go: Common routines for transports
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func createLogFile(logDir string, logFile string, configName string, dirMode os.FileMode, fileMode os.FileMode) (*os.File, string, error) {
	// If logfile is set, use it as the name of the log file. If not, generate
	// a standard name under logDir and create any intermediate directories as
	// needed.

	path := logFile
	if path == "" {
		dir := filepath.Join(logDir, configName)
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return nil, "", fmt.Errorf("Error trying to crete dir tree %q: %v", dir, err)
		}
		ymd := time.Now().Format("2006-01-02")
		path = filepath.Join(dir, configName+"-"+ymd+".log")
	}

	w, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, fileMode)
	if err != nil {
		return nil, path, fmt.Errorf("Error opening %q: %v", path, err)
	}
	return w, path, err
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
