package transports

// common.go: Common routines for transports
//
// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import (
	"os"
	"path/filepath"
	"time"
)

func createLogFile(logDir string, logFile string, configName string, dirMode os.FileMode, fileMode os.FileMode) (*os.File, string, error) {
	var (
		path string
	)

	// If logfile is set, use it as the name of the log file. If not, generate
	// a standard name under logDir and create any intermediate directories as
	// needed.
	if logFile == "" {
		dir := filepath.Join(logDir, configName)
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return nil, "", err
		}
		ymd := time.Now().Format("2006-01-02")
		path = filepath.Join(dir, configName+"-"+ymd+".log")
	}

	w, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, fileMode)
	if err != nil {
		return nil, path, err
	}
	return w, path, err
}
