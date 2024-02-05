// netbackup - Consistent multi-method backup tool
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2015-2024 by Marco Paganini <paganini AT paganini DOT net>

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
)

// writeNodeTextFile writes a record in a prometheus node-exporter
// compatible "textfile" format. The record is formatted as:
//
// backup{name="foobar", job="netbackup", status="success"} <timestamp>
//
// Existing lines with the same format and name will be overwritten.
// All other lines will remain intact.
//
// The function employs FLock() on a separate lockfile to prevent race
// conditions when modifying to the original file. All writes go into a
// temporary file that is atomically renamed to the final name once work is
// done.
func writeNodeTextFile(filename string, name string) error {
	lockfile := filename + ".lock"
	lock, err := os.Create(lockfile)
	if err != nil {
		return err
	}
	defer lock.Close()

	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	// Read contents from original filename.
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Rebuild output without any previous lines with the same name
	// and the new line added with the current unix timestamp.
	matchname, err := regexp.Compile(`backup[\s]*{.*name="` + name + `".*`)
	if err != nil {
		return err
	}

	output := []byte{}
	for _, line := range bytes.Split(data, []byte("\n")) {
		// See https://github.com/golang/go/issues/35130
		// To understand why this BS is needed here.
		if len(line) == 0 {
			continue
		}
		// Don't copy our own lines.
		if matchname.Match(line) {
			continue
		}
		output = append(output, line...)
		output = append(output, byte('\n'))
	}
	// Add our line.
	now := time.Now().Unix()
	s := fmt.Sprintf("backup{name=%q, job=\"netbackup\", status=\"success\"} %d\n", name, now)
	output = append(output, []byte(s)...)

	// Write to temporary file and rename it to the original file name.
	dirname, fname := filepath.Split(filename)
	if dirname == "" {
		dirname = "./"
	}

	temp, err := os.CreateTemp(dirname, fname)
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()

	_, err = temp.Write(output)
	if err != nil {
		return err
	}
	temp.Close()

	if err := os.Rename(temp.Name(), filename); err != nil {
		return err
	}

	return nil
}
