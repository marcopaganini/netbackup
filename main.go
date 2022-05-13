// netbackup - Consistent multi-method backup tool
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2015-2019 by Marco Paganini <paganini AT paganini DOT net>

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
)

const (
	progName     = "netbackup"
	devMapperDir = "/dev/mapper"

	// Default permissions for log directories and files.
	// The current umask will apply to these.
	defaultLogDirMode  = 0777
	defaultLogFileMode = 0666

	// External commands.
	mountCmd      = "mount"
	umountCmd     = "umount"
	cryptSetupCmd = "cryptsetup"
	fsckCmd       = "fsck"
	tunefsCmd     = "tune2fs"
)

var (
	// Build is filled by go build -ldflags during build.
	Build string

	// Generic logging object
	log *logger.Logger
)

// logPath constructs the name for the output log using the the name and
// the current system date.
func logPath(name string, logDir string) string {
	ymd := time.Now().Format("2006-01-02")
	dir := filepath.Join(logDir, name)
	return filepath.Join(dir, progName+"-"+name+"."+ymd+".log")
}

// logOpen opens (for append) or creates (if needed) the specified file.
// If the file doesn't exist, all intermediate directories will be created.
// Returns an *os.File to the just opened file.
func logOpen(path string) (*os.File, error) {
	// Create full directory path if it doesn't exist yet.
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, defaultLogDirMode); err != nil {
			return nil, fmt.Errorf("unable to create dir tree %q: %v", dir, err)
		}
	}

	// Open for append or create if doesn't exist.
	w, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, defaultLogFileMode)
	if err != nil {
		return nil, fmt.Errorf("unable to open %q: %v", path, err)
	}
	return w, nil
}

// isMounted returns true if the specified directory is mounted, false otherwise.
// This function needs /proc/mounts to work.
func isMounted(dirname string) (bool, error) {
	d, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return false, err
	}
	cslice := strings.Split(string(d), "\n")
	for _, line := range cslice {
		f := strings.Split(line, " ")
		if len(f) > 1 && f[1] == dirname {
			return true, nil
		}
	}
	return false, nil
}

func main() {
	ctx := context.Background()
	log = logger.New("")

	// Parse command line flags and read config file.
	if err := parseFlags(); err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	// If version request, just print version and exit.
	if opt.version {
		fmt.Printf("Version (Build): %s\n", Build)
		os.Exit(0)
	}

	// Open and parse config file.
	cfg, err := os.Open(opt.config)
	if err != nil {
		log.Fatalf("Unable to open config file: %v\n", err)
	}
	config, err := config.ParseConfig(cfg)
	if err != nil {
		log.Fatalf("Configuration error in %q: %v\n", opt.config, err)
	}

	// Set log output and all other log related parameters.
	verbose := int(opt.verbose)
	if verbose > 0 {
		log.SetVerboseLevel(verbose)
	}
	// Create output log. Use the name specified in the config, if any,
	// or create a "standard" name using the backup name and date.
	logFilename := config.Logfile
	if logFilename == "" {
		logFilename = logPath(config.Name, config.LogDir)
	}
	outLog, err := logOpen(logFilename)
	if err != nil {
		log.Fatalf("Unable to open/create logfile: %v\n", err)
	}
	defer outLog.Close()

	// Configure log to log everything to stderr and outLog
	log.SetMirrorOutput(outLog)

	// Add Logger to context.
	ctx = logger.WithLogger(ctx, log)

	if opt.dryrun {
		log.Verboseln(1, "Warning: Dry-Run mode. Won't execute any commands.")
	}

	// Create new Backup and execute.
	b := NewBackup(config, opt.dryrun)

	if err = b.Run(ctx); err != nil {
		log.Fatalln(err)
	}
	log.Verboseln(1, "*** Backup Result: Success")
}
