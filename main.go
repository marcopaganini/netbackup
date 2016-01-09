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

	// Output Log
	outLog *os.File = os.Stderr
)

// NewBackup creates a new Backup instance.
func NewBackup(log *logger.Logger, config *config.Config, outLog *os.File, verbose int, dryRun bool) *Backup {
	// Create new Backup and execute.
	return &Backup{
		log:     log,
		config:  config,
		outLog:  outLog,
		verbose: verbose,
		dryRun:  opt.dryrun}
}

// mountDev mounts the destination device into a temporary mount point and
// returns the mount point name.
func (b *Backup) mountDev() (string, error) {
	tmpdir, err := ioutil.TempDir("", "netbackup_mount")
	if err != nil {
		return "", fmt.Errorf("unable to create temp directory: %v", err)
	}

	// We use the mount command instead of the mount syscall as it makes
	// simpler to specify defaults in /etc/fstab.
	cmd := mountCmd + " " + b.config.DestDev + " " + tmpdir
	if err := runCommand("MOUNT", cmd, nil); err != nil {
		return "", err
	}

	return tmpdir, nil
}

// umountDev dismounts the destination device specified in config.DestDev.
func (b *Backup) umountDev() error {
	cmd := umountCmd + " " + b.config.DestDev
	return runCommand("UMOUNT", cmd, nil)
}

// openLuks opens the luks destination device into a temporary /dev/mapper
// device file and retuns the /dev/mapper device filename.
func (b *Backup) openLuks() (string, error) {
	// Our temporary dev/mapper device is based on the config name
	devname := "netbackup_" + b.config.Name
	devfile := filepath.Join(devMapperDir, devname)

	// Make sure it doesn't already exist
	if _, err := os.Stat(devfile); err == nil {
		return "", fmt.Errorf("device mapper file %q already exists", devfile)
	}

	// cryptsetup LuksOpen
	cmd := cryptSetupCmd
	if b.config.LuksKeyFile != "" {
		cmd += " --key-file " + b.config.LuksKeyFile
	}
	cmd += " luksOpen " + b.config.LuksDestDev + " " + devname
	if err := runCommand("LUKS_OPEN", cmd, nil); err != nil {
		return "", err
	}

	return devfile, nil
}

// closeLuks closes the current destination device.
func (b *Backup) closeLuks() error {
	// cryptsetup luksClose needs the /dev/mapper device name.
	cmd := cryptSetupCmd + " luksClose " + b.config.DestDev
	return runCommand("LUKS_CLOSE", cmd, nil)
}

// cleanFilesystem runs fsck to make sure the filesystem under config.dest_dev is
// intact, and sets the number of times to check to 0 and the last time
// checked to now. This option should only be used in EXTn filesystems or
// filesystems that support tunefs.
func (b *Backup) cleanFilesystem() error {
	// fsck (read-only check)
	cmd := fsckCmd + " -n " + b.config.DestDev
	if err := runCommand("FS_CLEANUP", cmd, nil); err != nil {
		return fmt.Errorf("error running %q: %v", cmd, err)
	}
	// Tunefs
	cmd = tunefsCmd + " -C 0 -T now " + b.config.DestDev
	return runCommand("FS_CLEANUP", cmd, nil)
}

// Run executes the backup according to the config file and options.
func (b *Backup) Run() error {
	var transp interface {
		Run() error
	}

	if !b.dryRun {
		// Open LUKS device, if needed
		if b.config.LuksDestDev != "" {
			devfile, err := b.openLuks()
			if err != nil {
				return fmt.Errorf("Error opening LUKS device %q: %v", b.config.LuksDestDev, err)
			}
			// Set the destination device to the /dev/mapper device opened by
			// LUKS. This should allow the natural processing to mount and
			// dismount this device.
			b.config.DestDev = devfile

			// close luks device at the end
			defer b.closeLuks()
			defer time.Sleep(2 * time.Second)
		}

		// Run cleanup on fs prior to backup, if requested.
		if b.config.FSCleanup {
			if err := b.cleanFilesystem(); err != nil {
				return fmt.Errorf("Error performing pre-backup cleanup on %q: %v", b.config.DestDev, err)
			}
		}

		// Mount destination device, if needed.
		if b.config.DestDev != "" {
			tmpdir, err := b.mountDev()
			if err != nil {
				return fmt.Errorf("Error opening destination device %q: %v", b.config.DestDev, err)
			}
			// After we mount the destination device, we set Destdir to that location
			// so the backup will proceed seamlessly.
			b.config.DestDir = tmpdir

			// umount destination filesystem and remove temp mount point.
			defer os.Remove(b.config.DestDir)
			defer b.umountDev()
			// For some reason, not having a pause before attempting to unmount
			// can generate a race condition where umount complains that the fs
			// is busy (even though the transport is already down.)
			defer time.Sleep(2 * time.Second)
		}
	}

	var err error

	// Create new transport based on config.Transport
	switch b.config.Transport {
	case "rclone":
		transp, err = transports.NewRcloneTransport(b.config, nil, b.outLog, b.verbose, b.dryRun)
	case "rdiff-backup":
		transp, err = transports.NewRdiffBackupTransport(b.config, nil, b.outLog, b.verbose, b.dryRun)
	case "rsync":
		transp, err = transports.NewRsyncTransport(b.config, nil, b.outLog, b.verbose, b.dryRun)
	default:
		return fmt.Errorf("Unknown transport %q", b.config.Transport)
	}
	if err != nil {
		return fmt.Errorf("Error creating %s transport: %v", b.config.Transport, err)
	}

	// Execute pre-commands, if any.
	if b.config.PreCommand != "" && !b.dryRun {
		if err := runCommand("PRE", b.config.PreCommand, nil); err != nil {
			return fmt.Errorf("Error running pre-command: %v", err)
		}
	}

	// Make it so...
	if err := transp.Run(); err != nil {
		return fmt.Errorf("Error running backup: %v", err)
	}

	// Execute post-commands, if any.
	if b.config.PostCommand != "" && !b.dryRun {
		if err := runCommand("POST", b.config.PostCommand, nil); err != nil {
			return fmt.Errorf("Error running post-command (possible backup failure): %v", err)
		}
	}

	return nil
}

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

// runCommand executes the given command using the shell. A prefix will
// be used to log the commands to the output log. Returns error.
func runCommand(prefix string, cmd string, ex *execute.Execute) error {
	m := fmt.Sprintf("%s Command: %q", prefix, cmd)
	log.Verboseln(1, m)

	// Create a new execute object, if current is nil
	e := ex
	if e == nil {
		e = execute.New()
	}

	// All streams copied to output log with "PRE:" as a prefix.
	e.SetStdout(func(buf string) error { _, err := fmt.Fprintf(outLog, "%s(stdout): %s\n", prefix, buf); return err })
	e.SetStderr(func(buf string) error { _, err := fmt.Fprintf(outLog, "%s(stderr): %s\n", prefix, buf); return err })

	// Run using shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	err := e.Exec([]string{shell, "-c", "--", cmd})
	if err != nil {
		errmsg := fmt.Sprintf("%s returned: %v", prefix, err)
		log.Verbosef(1, "*** %s\n", errmsg)
		return fmt.Errorf(errmsg)
	}
	log.Verbosef(1, "%s returned: OK\n", prefix)
	return nil
}

// logPath constructs the name for the output log using the the name and
// the current system date.
func logPath(name string, logDir string) string {
	ymd := time.Now().Format("2006-01-02")
	dir := filepath.Join(logDir, name)
	return filepath.Join(dir, name+"-"+ymd+".log")
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

// main
func main() {
	log = logger.New("")

	// Parse command line flags and read config file.
	if err := parseFlags(); err != nil {
		log.Printf("Command line error: %v", err)
		os.Exit(osError)
	}

	// Set verbose level
	verbose := int(opt.verbose)
	if verbose > 0 {
		log.SetVerboseLevel(verbose)
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

	// Create output log. Use the name specified in the config, if any,
	// or create a "standard" name using the backup name and date.
	logFilename := config.Logfile
	if logFilename == "" {
		logFilename = logPath(config.Name, defaultLogDir)
	}
	outLog, err := logOpen(logFilename)
	if err != nil {
		log.Printf("Unable to open/create logfile: %v", err)
		os.Exit(osError)
	}
	defer outLog.Close()

	// Configure log to log everything to stderr and outLog
	log.SetOutput([]*os.File{os.Stderr, outLog})

	// Create new Backup and execute.
	b := NewBackup(log, config, outLog, verbose, opt.dryrun)

	if err = b.Run(); err != nil {
		log.Println(err)
		os.Exit(osError)
	}
	log.Verboseln(1, "*** Backup Result: Success")
	os.Exit(osSuccess)
}
