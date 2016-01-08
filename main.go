// netbackup - Consistent multi-method backup tool
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

package main

import (
	"flag"
	"fmt"
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
	defaultLogDir = "/tmp/log/netbackup"
	devMapperDir  = "/dev/mapper"

	// Default permissions for log directories and files.
	// The current umask will apply to these.
	defaultLogDirMode  = 0777
	defaultLogFileMode = 0666

	// Return codes
	osSuccess = 0
	osError   = 1

	// External commands.
	mountCmd      = "mount"
	umountCmd     = "umount"
	cryptSetupCmd = "cryptsetup"
	fsckCmd       = "fsck"
	tunefsCmd     = "tune2fs"
	dfCmd         = "df"
)

// Backup contains information for a given backup instance.
type Backup struct {
	log     *logger.Logger
	config  *config.Config
	outLog  *os.File
	verbose int
	dryRun  bool
}

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

// NewBackup creates a new Backup instance.
func NewBackup(log *logger.Logger, config *config.Config, verbose int, dryRun bool) *Backup {
	// Create new Backup and execute.
	return &Backup{
		log:     log,
		config:  config,
		outLog:  os.Stdout,
		verbose: verbose,
		dryRun:  opt.dryrun}
}

// runCommand executes the given command using the shell. A prefix will
// be used to log the commands to the output log. Returns error.
func (b *Backup) runCommand(prefix string, cmd string, ex *execute.Execute) error {
	m := fmt.Sprintf("%s Command: %q", prefix, cmd)
	fmt.Fprintf(b.outLog, "%s\n", m)
	b.log.Verboseln(int(opt.verbose), m)

	// Create a new execute object, if current is nil
	e := ex
	if e == nil {
		e = execute.New()
	}

	// All streams copied to output log with "PRE:" as a prefix.
	e.SetStdout(func(buf string) error { _, err := fmt.Fprintf(b.outLog, "%s(stdout): %s\n", prefix, buf); return err })
	e.SetStderr(func(buf string) error { _, err := fmt.Fprintf(b.outLog, "%s(stderr): %s\n", prefix, buf); return err })

	// Run using shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	err := e.Exec([]string{shell, "-c", "--", cmd})
	if err != nil {
		errmsg := fmt.Sprintf("%s returned: %v", prefix, err)
		fmt.Fprintf(b.outLog, "*** %s\n", errmsg)
		return fmt.Errorf(errmsg)
	}
	fmt.Fprintf(b.outLog, "%s returned: OK\n", prefix)
	return nil
}

// createOutputLog creates a new output log file. If config.LogFile is set, it
// is used unchanged. If not, a new log is created based under logDir using the
// configuration name, and the system date. Intermediate directories are
// created as needed. Sets b.config.outLog pointing to the writer of the log
// just created. Returns the name of the file and error.
func (b *Backup) createOutputLog(logDir string) error {
	path := b.config.Logfile
	if path == "" {
		dir := filepath.Join(logDir, b.config.Name)
		if err := os.MkdirAll(dir, defaultLogDirMode); err != nil {
			return fmt.Errorf("unable to create dir tree %q: %v", dir, err)
		}
		ymd := time.Now().Format("2006-01-02")
		path = filepath.Join(dir, b.config.Name+"-"+ymd+".log")
	}

	w, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, defaultLogFileMode)
	if err != nil {
		return fmt.Errorf("unable to open %q: %v", path, err)
	}
	b.outLog = w
	return err
}

// mountDestDev mounts the destination device specified in b.config.DestDev into
// a temporary mount point and set b.config.DestDir to point to this directory.
func (b *Backup) mountDestDev() error {
	tmpdir, err := ioutil.TempDir("", "netbackup_mount")
	if err != nil {
		return fmt.Errorf("unable to create temp directory: %v", err)
	}

	// We use the mount command instead of the mount syscal as it makes
	// simpler to specify defaults in /etc/fstab.
	cmd := mountCmd + " " + b.config.DestDev + " " + tmpdir
	if err := b.runCommand("MOUNT", cmd, nil); err != nil {
		return err
	}

	b.config.DestDir = tmpdir
	return nil
}

// umountDestDev dismounts the destination device specified in config.DestDev.
func (b *Backup) umountDestDev() error {
	cmd := umountCmd + " " + b.config.DestDev
	return b.runCommand("UMOUNT", cmd, nil)
}

// openLuksDestDev opens the luks device specified by config.LuksDestDev and sets
// b.config.DestDev to the /dev/mapper device.
func (b *Backup) openLuksDestDev() error {
	// Our temporary dev/mapper device is based on the config name
	devname := "netbackup_" + b.config.Name
	devfile := filepath.Join(devMapperDir, devname)

	// Make sure it doesn't already exist
	if _, err := os.Stat(devfile); err == nil {
		return fmt.Errorf("device mapper file %q already exists", devfile)
	}

	// cryptsetup LuksOpen
	cmd := cryptSetupCmd
	if b.config.LuksKeyFile != "" {
		cmd += " --key-file " + b.config.LuksKeyFile
	}
	cmd += " luksOpen " + b.config.LuksDestDev + " " + devname
	if err := b.runCommand("LUKS_OPEN", cmd, nil); err != nil {
		return err
	}

	// Set the destination device to devfile so the normal processing
	// will be sufficient to mount and dismount this device.
	b.config.DestDev = devfile
	return nil
}

// closeLuksDestDev closes the luks device specified by b.config.LuksDestDev.
func (b *Backup) closeLuksDestDev() error {
	// Note that even though this function is called closeLuksDestDev we use
	// the mount point under /dev/mapper to close the device.  The mount point
	// was previously set by openLuksDestDev.
	cmd := cryptSetupCmd + " luksClose " + b.config.DestDev
	return b.runCommand("LUKS_CLOSE", cmd, nil)
}

// fsCleanup runs fsck to make sure the filesystem under config.dest_dev is
// intact, and sets the number of times to check to 0 and the last time
// checked to now. This option should only be used in EXTn filesystems or
// filesystems that support tunefs.
func (b *Backup) fsCleanup() error {
	// fsck (read-only check)
	cmd := fsckCmd + " -n " + b.config.DestDev
	if err := b.runCommand("FS_CLEANUP", cmd, nil); err != nil {
		return fmt.Errorf("error running %q: %v", cmd, err)
	}
	// Tunefs
	cmd = tunefsCmd + " -C 0 -T now " + b.config.DestDev
	return b.runCommand("FS_CLEANUP", cmd, nil)
}

// Run executes the backup according to the config file and options.
func (b *Backup) Run() error {
	var transp interface {
		Run() error
	}

	// Open or create the output log file. This log will contain a transcript
	// of stdout and stderr from all commands executed by this program.
	err := b.createOutputLog(defaultLogDir)
	if err != nil {
		return fmt.Errorf("Error creating output log: %v", err)
	}
	defer b.outLog.Close()

	if !b.dryRun {
		// Open LUKS device, if needed
		if b.config.LuksDestDev != "" {
			if err := b.openLuksDestDev(); err != nil {
				return fmt.Errorf("Error opening LUKS device %q: %v", b.config.LuksDestDev, err)
			}
			// close luks device at the end
			defer b.closeLuksDestDev()
			defer time.Sleep(2 * time.Second)
		}

		// Run cleanup on fs prior to backup, if requested.
		if b.config.FSCleanup {
			if err := b.fsCleanup(); err != nil {
				return fmt.Errorf("Error performing pre-backup cleanup on %q: %v", b.config.DestDev, err)
			}
		}

		// Mount destination device, if needed.
		if b.config.DestDev != "" {
			if err := b.mountDestDev(); err != nil {
				return fmt.Errorf("Error opening destination device %q: %v", b.config.DestDev, err)
			}
			// umount destination filesystem and remove temp mount point.
			defer os.Remove(b.config.DestDir)
			defer b.umountDestDev()
			// For some reason, not having a pause before attempting to unmount
			// can generate a race condition where umount complains that the fs
			// is busy (even though the transport is already down.)
			defer time.Sleep(2 * time.Second)
		}
	}

	// Create new transport based on config.Transport
	switch b.config.Transport {
	case "rclone":
		transp, err = transports.NewRcloneTransport(b.config, nil, b.outLog, int(opt.verbose), b.dryRun)
	case "rdiff-backup":
		transp, err = transports.NewRdiffBackupTransport(b.config, nil, b.outLog, int(opt.verbose), b.dryRun)
	case "rsync":
		transp, err = transports.NewRsyncTransport(b.config, nil, b.outLog, int(opt.verbose), b.dryRun)
	default:
		return fmt.Errorf("Unknown transport %q", b.config.Transport)
	}
	if err != nil {
		return fmt.Errorf("Error creating %s transport: %v", b.config.Transport, err)
	}

	// Execute pre-commands, if any.
	if b.config.PreCommand != "" && !b.dryRun {
		if err := b.runCommand("PRE", b.config.PreCommand, nil); err != nil {
			return fmt.Errorf("Error running pre-command: %v", err)
		}
	}

	// Make it so...
	if err := transp.Run(); err != nil {
		return fmt.Errorf("Error running backup: %v", err)
	}
	fmt.Fprintf(b.outLog, "*** Backup Result: Success\n")

	// Execute post-commands, if any.
	if b.config.PostCommand != "" && !b.dryRun {
		if err := b.runCommand("POST", b.config.PostCommand, nil); err != nil {
			fmt.Fprintf(b.outLog, "*** Backup Result: Failure (%v)\n", err)
			return fmt.Errorf("Error running post-command: %v", err)
		}
	}

	return nil
}

// main
func main() {
	log = logger.New("")

	// Parse command line flags and read config file.
	if err := parseFlags(); err != nil {
		log.Printf("Command line error: %v", err)
		os.Exit(osError)
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
		os.Exit(osError)
	}
	config, err := config.ParseConfig(cfg)
	if err != nil {
		log.Printf("Configuration error in %q: %v", opt.config, err)
		os.Exit(osError)
	}

	// Create new Backup and execute.
	b := NewBackup(log, config, int(opt.verbose), opt.dryrun)

	if err = b.Run(); err != nil {
		log.Println(err)
		os.Exit(osError)
	}

	os.Exit(osSuccess)
}
