// netbackup - Consistent multi-method backup tool
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/transports"
)

const (
	// DEBUG
	defaultLogDir      = "/tmp/log/netbackup"
	defaultLogDirMode  = 0770
	defaultLogFileMode = 0660
)

var (
	// Generic logging object
	log *logger.Logger
)

// usage prints an error message and program usage to stderr, exiting after
// that.
func usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
	}
	fmt.Fprintf(os.Stderr, "Usage%s:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

// createOutputLog creates a new output logfile. If logFile is set, it is used
// unchanged. If not, a new log is created based on the default log directory,
// the configuration name, and the system date. Intermediate directories are
// created as needed. Returns a *os.File, the file created, error.
func createOutputLog(logFile string, configName string) (*os.File, string, error) {
	path := logFile
	if path == "" {
		dir := filepath.Join(defaultLogDir, configName)
		if err := os.MkdirAll(dir, defaultLogDirMode); err != nil {
			return nil, "", fmt.Errorf("Error trying to crete dir tree %q: %v", dir, err)
		}
		ymd := time.Now().Format("2006-01-02")
		path = filepath.Join(dir, configName+"-"+ymd+".log")
	}

	w, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, defaultLogFileMode)
	if err != nil {
		return nil, path, fmt.Errorf("Error opening %q: %v", path, err)
	}
	return w, path, err
}

// Transport constructor function definition
type transportFunc func(*config.Config, transports.CommandRunner, io.Writer, int, bool) (*transports.Transport, error)

func main() {
	// List of transport names and their constructors
	TransportList := map[string]transportFunc{
		"rclone": transports.NewRcloneTransport,
	}

	log = logger.New("")

	// Parse command line flags and read config file.
	if err := parseFlags(); err != nil {
		log.Fatalf("Error: %v", err)
	}
	// Set verbose level
	if opt.verbose > 0 {
		log.SetVerboseLevel(int(opt.verbose))
	}
	if opt.dryrun {
		log.Verbosef(2, "Warning: Dry-Run mode. Won't execute any commands.")
	}

	cfg, err := os.Open(opt.config)
	if err != nil {
		log.Fatalf("Unable to open config file: %v", err)
	}

	config, err := config.ParseConfig(cfg)
	if err != nil {
		log.Fatalf("Configuration error in %q: %v", opt.config, err)
	}

	// Make sure the requested transport is valid. Exit with an error if not.
	tnew, found := TransportList[config.Transport]
	if !found {
		log.Fatalf("Unknown transport requested: %q", config.Transport)
	}

	// Open output log file
	outLog, outPath, err := createOutputLog(config.Logfile, config.Name)
	if err != nil {
		log.Fatalf("Error opening output logfile: %q: %v", outPath, err)
	}
	defer outLog.Close()

	// Create a new transport and execute backup.
	t, err := tnew(config, nil, outLog, int(opt.verbose), opt.dryrun)
	if err != nil {
		log.Fatalf("Error creating %s transport: %v", config.Transport, err)
	}
	if err := t.Run(); err != nil {
		log.Fatalln(err)
	}
}
