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
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/execute"
	"github.com/marcopaganini/netbackup/transports"
)

const (
	// DEBUG
	defaultLogDir = "/tmp/log/netbackup"

	// Default permissions for log directories and files.
	// The current umask will apply to these.
	defaultLogDirMode  = 0777
	defaultLogFileMode = 0666

	// Return codes
	osSuccess = 0
	osError   = 1

	// External commands.
	mountCmd  = "mount"
	umountCmd = "umount"
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

// createOutputLog creates a new output log file. If logPath is set, it is used
// unchanged. If not, a new log is created based under logDir using the
// configuration name, and the system date. Intermediate directories are
// created as needed. Returns a *os.File, the file created, error.
func createOutputLog(logPath string, logDir string, configName string) (*os.File, string, error) {
	path := logPath
	if path == "" {
		dir := filepath.Join(logDir, configName)
		if err := os.MkdirAll(dir, defaultLogDirMode); err != nil {
			return nil, "", fmt.Errorf("Error trying to create dir tree %q: %v", dir, err)
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

// shellRun run a command string using the shell using the specified execute object.
func shellRun(e *execute.Execute, cmd string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	a := []string{shell, "-c", "--", cmd}
	return e.Exec(a)
}

// runCommand executes the pre or post commands using the shell. A prefix will
// be used to log the commands to the output log (usually, "PRE" for
// pre-commands or "POST" for post-commands Returns error.
func runCommand(prefix string, cmd string, ex *execute.Execute, outLog io.Writer) error {
	m := fmt.Sprintf("%s Command: %q", prefix, cmd)
	log.Verboseln(int(opt.verbose), m)
	if opt.dryrun {
		return nil
	}

	// Create a new execute object, if current is nil
	e := ex
	if e == nil {
		e = execute.New()
	}

	// All streams copied to output log with "PRE:" as a prefix.
	e.SetStdout(func(buf string) error { _, err := fmt.Fprintf(outLog, "%s(stdout): %s\n", prefix, buf); return err })
	e.SetStderr(func(buf string) error { _, err := fmt.Fprintf(outLog, "%s(stderr): %s\n", prefix, buf); return err })

	fmt.Fprintf(outLog, "*** %s\n", m)
	err := shellRun(e, cmd)
	fmt.Fprintf(outLog, "*** %s returned: %v\n", prefix, err)

	return err
}

// mountDestDev mounts the destination device specified in config.DestDev into
// a temporary mount point and set config.DestDir to point to this directory.
func mountDestDev(config *config.Config, outLog io.Writer) error {
	tmpdir, err := ioutil.TempDir("", "netbackup_mount")
	if err != nil {
		return fmt.Errorf("unable to create temp directory: %v", err)
	}

	// We use the mount command instead of the mount syscal as it makes
	// simpler to specify defaults in /etc/fstab.
	cmd := mountCmd + " " + config.DestDev + " " + tmpdir
	if err := runCommand("MOUNT", cmd, nil, outLog); err != nil {
		return fmt.Errorf("error running %q: %v", cmd, err)
	}

	config.DestDir = tmpdir
	return nil
}

// umountDestDev dismounts the destination device specified in config.DestDev.
func umountDestDev(config *config.Config, outLog io.Writer) error {
	cmd := umountCmd + " " + config.DestDev
	if err := runCommand("UMOUNT", cmd, nil, outLog); err != nil {
		return fmt.Errorf("error running %q: %v", cmd, err)
	}
	return nil
}

func backup() int {
	var transp interface {
		Run() error
	}
	log = logger.New("")

	// Parse command line flags and read config file.
	if err := parseFlags(); err != nil {
		log.Printf("Error: %v", err)
		return osError
	}
	// Set verbose level
	if opt.verbose > 0 {
		log.SetVerboseLevel(int(opt.verbose))
	}
	if opt.dryrun {
		log.Verbosef(2, "Warning: Dry-Run mode. Won't execute any commands.")
	}

	// Open and parse config file
	cfg, err := os.Open(opt.config)
	if err != nil {
		log.Printf("Unable to open config file: %v", err)
		return osError
	}
	config, err := config.ParseConfig(cfg)
	if err != nil {
		log.Printf("Configuration error in %q: %v", opt.config, err)
		return osError
	}

	// Open or create the output log file. This log will contain a transcript
	// of stdout and stderr from all commands executed by this program.
	outLog, outPath, err := createOutputLog(config.Logfile, defaultLogDir, config.Name)
	if err != nil {
		log.Printf("Error opening output logfile: %q: %v", outPath, err)
		return osError
	}
	defer outLog.Close()

	// Mount destination device, if needed.
	if config.DestDev != "" && !opt.dryrun {
		if err := mountDestDev(config, outLog); err != nil {
			log.Printf("Error opening destination device %q: %v", config.DestDev, err)
			return osError
		}
		// umount destination filesystem and remove temp mount point.
		defer os.Remove(config.DestDir)
		defer umountDestDev(config, outLog)
		// For some reason, not having a pause before attempting to unmount
		// can generate a race condition where umount complains that the fs
		// is busy (even though the transport is already down.)
		defer time.Sleep(2 * time.Second)
	}

	// Create new transport based on config.Transport
	switch config.Transport {
	case "rclone":
		transp, err = transports.NewRcloneTransport(config, nil, outLog, int(opt.verbose), opt.dryrun)
	case "rdiff-backup":
		transp, err = transports.NewRdiffBackupTransport(config, nil, outLog, int(opt.verbose), opt.dryrun)
	case "rsync":
		transp, err = transports.NewRsyncTransport(config, nil, outLog, int(opt.verbose), opt.dryrun)
	default:
		log.Printf("Unknown transport %q", config.Transport)
		return osError
	}
	if err != nil {
		log.Printf("Error creating %s transport: %v", config.Transport, err)
		return osError
	}

	// Execute pre-commands, if any.
	if err := runCommand("PRE", config.PreCommand, nil, outLog); err != nil {
		log.Printf("Error running pre-command: %v", err)
		return osError
	}

	// Make it so...
	if err := transp.Run(); err != nil {
		log.Printf("Error running backup: %v", err)
		return osError
	}
	fmt.Fprintf(outLog, "*** Backup Result: Success\n")

	// Execute post-commands, if any.
	if err := runCommand("POST", config.PostCommand, nil, outLog); err != nil {
		fmt.Fprintf(outLog, "*** Backup Result: Failure (%v)\n", err)
		log.Printf("Error running post-command: %v", err)
		return osError
	}

	return osSuccess
}

func main() {
	os.Exit(backup())
}
