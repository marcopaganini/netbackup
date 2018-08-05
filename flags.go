// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015-2018 by Marco Paganini <paganini AT paganini DOT net>

package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	// Flag defaults
	defaultOptVerboseLevel = 0
	defaultOptDryRun       = false
)

type multiLevelInt int

type cmdLineOpts struct {
	config  string
	dryrun  bool
	verbose multiLevelInt
}

var (
	// Command line Flags
	opt cmdLineOpts
)

// Definitions for the custom flag type multiLevelInt

// Return the string representation of the flag.
// The String method's output will be used in diagnostics.
func (m *multiLevelInt) String() string {
	return fmt.Sprint(*m)
}

// Increase the value of multiLevelInt. This accepts multiple values
// and sets the variable to the number of times those values appear in
// the command-line. Useful for "verbose" and "Debug" levels.
func (m *multiLevelInt) Set(_ string) error {
	*m++
	return nil
}

// Behave as a bool (i.e. no arguments)
func (m *multiLevelInt) IsBoolFlag() bool {
	return true
}

// Parse the command line and set the global opt variable. Return error
// if the basic sanity checking of flags fails.
func parseFlags() error {
	// Parse command line
	flag.StringVar(&opt.config, "config", "", "Config File")
	flag.StringVar(&opt.config, "c", "", "Config File (shorthand)")
	flag.BoolVar(&opt.dryrun, "dry-run", defaultOptDryRun, "Dry-run mode")
	flag.BoolVar(&opt.dryrun, "n", defaultOptDryRun, "Dry-run mode (shorthand)")
	flag.Var(&opt.verbose, "verbose", "Verbose mode (use multiple times to increase level)")
	flag.Var(&opt.verbose, "v", "Verbose mode (use multiple times to increase level)")
	flag.Parse()

	// Config is mandatory
	if opt.config == "" {
		usage()
		return fmt.Errorf("Configuration file must be specified with --config=config_filename")
	}
	return nil
}

// returns a formatted error message including the program's usage.
func usage() {
	fmt.Printf("Usage %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Println("")
}
