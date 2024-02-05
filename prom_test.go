// This file is part of netbackup, a frontend to simplify periodic backups.
// For further information, check https://github.com/marcopaganini/netbackup
//
// (C) 2015-2024 by Marco Paganini <paganini AT paganini DOT net>

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Number of records to create/test.
const numRecords = 20

// generate creates multiple node compatible backup records in parallel.
func generate(tmpfile string, ch chan error) {
	// Generate multiple backup records.
	for i := 0; i < numRecords; i++ {
		go func(ch chan error, name string) {
			err := writeNodeTextFile(tmpfile, name)
			ch <- err
		}(ch, fmt.Sprintf("backup%03.3d", i))
	}
}

// errcheck returns the first error found in a slice of error channels.
func errcheck(ch chan error) error {
	var saved error
	for i := 0; i < numRecords; i++ {
		err := <-ch
		if err != nil && saved != nil {
			saved = err
		}
	}
	return saved
}

// filecheck parses the generated file and makes sure we have exactly
// numRecords properly formatted records.
func filecheck(t *testing.T, tmpfile string) error {
	data, err := os.ReadFile(tmpfile)
	if err != nil {
		return err
	}
	lines := bytes.Split(data, []byte("\n"))

	t.Log("Generated file contents")
	for i, v := range lines {
		t.Logf("%d: %s\n", i, v)
	}

	// Make sure we have exactly numRecords lines.
	numlines := len(lines) - 1 // Skip the last blank line caused by a newline.
	if numlines != numRecords {
		return fmt.Errorf("number of lines mismatch: expected %d, found %d", numRecords, numlines)
	}

	// Fill in the "names" map with all names found in the file.
	re := regexp.MustCompile(`backup[\s]*{name="([^"]*)", job="netbackup", status="success"} [0-9]+`)
	names := map[string]bool{}
	for _, line := range lines {
		// Skip blank line at the end.
		if len(line) == 0 {
			continue
		}
		matches := re.FindSubmatch(line)
		if matches != nil {
			names[string(matches[1])] = true
		}
	}

	// Make sure all names are present.
	missing := []string{}
	for i := 0; i < numRecords; i++ {
		name := fmt.Sprintf("backup%03.3d", i)
		_, ok := names[name]
		if !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing backup names in output: %s", strings.Join(missing, ", "))
	}

	return nil
}

func TestMulti(t *testing.T) {
	ch := make(chan error, numRecords)

	tmpfile := filepath.Join(t.TempDir(), "testfile")
	generate(tmpfile, ch)

	if err := errcheck(ch); err != nil {
		t.Errorf("TestMulti: error writing textfile: %v", err)
	}
	if err := filecheck(t, tmpfile); err != nil {
		t.Errorf("TestMulti: file contents error: %v", err)
	}
}
