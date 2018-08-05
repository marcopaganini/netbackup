// netbackup - Consistent multi-method backup tool
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2015-2018 by Marco Paganini <paganini AT paganini DOT net>

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/execute"
	"github.com/marcopaganini/netbackup/transports"
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
	dfCmd         = "df"
)

// Backup contains information for a given backup instance.
type Backup struct {
	log    *logger.Logger
	config *config.Config
	dryRun bool
}

var (
	// Generic logging object
	log *logger.Logger
)

// NewBackup creates a new Backup instance.
func NewBackup(config *config.Config, log *logger.Logger, dryRun bool) *Backup {
	// Create new Backup and execute.
	return &Backup{
		log:    log,
		config: config,
		dryRun: opt.dryrun}
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
	cmd := []string{mountCmd, b.config.DestDev, tmpdir}
	if err := execute.Run("MOUNT", cmd, b.log); err != nil {
		return "", err
	}

	return tmpdir, nil
}

// umountDev dismounts the destination device specified in config.DestDev.
func (b *Backup) umountDev() error {
	cmd := []string{umountCmd, b.config.DestDev}
	return execute.Run("UMOUNT", cmd, b.log)
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
	cmd := []string{cryptSetupCmd}
	if b.config.LuksKeyFile != "" {
		cmd = append(cmd, "--key-file="+b.config.LuksKeyFile)
	}
	cmd = append(cmd, "luksOpen")
	cmd = append(cmd, b.config.LuksDestDev)
	cmd = append(cmd, devname)

	if err := execute.Run("LUKS_OPEN", cmd, b.log); err != nil {
		return "", err
	}

	return devfile, nil
}

// closeLuks closes the current destination device.
func (b *Backup) closeLuks() error {
	// cryptsetup luksClose needs the /dev/mapper device name.
	cmd := []string{cryptSetupCmd, "luksClose", b.config.DestDev}
	return execute.Run("LUKS_CLOSE", cmd, b.log)
}

// cleanFilesystem runs fsck to make sure the filesystem under config.dest_dev is
// intact, and sets the number of times to check to 0 and the last time
// checked to now. This option should only be used in EXTn filesystems or
// filesystems that support tunefs.
func (b *Backup) cleanFilesystem() error {
	// fsck (read-only check)
	cmd := []string{fsckCmd, "-n", b.config.DestDev}
	if err := execute.Run("FS_CLEANUP", cmd, b.log); err != nil {
		return fmt.Errorf("error running %q: %v", cmd, err)
	}
	// Tunefs
	cmd = []string{tunefsCmd, "-C", "0", "-T", "now", b.config.DestDev}
	return execute.Run("FS_CLEANUP", cmd, b.log)
}

// Run executes the backup according to the config file and options.
func (b *Backup) Run() error {
	var transp interface {
		Run() error
	}

	// If we're running in dry-run mode, we set dummy values for DestDev if
	// LuksDestDev is present, and for DestDir if DestDev is present. This hack
	// is necessary because these values won't be set to the appropriate values
	// in dry-run mode (since we don't want to open the luks destination in
	// that case) and the transports won't be able to show a full command line
	// in that case.

	if b.dryRun {
		if b.config.LuksDestDev != "" {
			b.config.DestDev = "dummy_dest_dev"
		}
		if b.config.DestDev != "" {
			b.config.DestDir = "dummy_dest_dir"
		}
	}

	if !b.dryRun {
		// Make sure sourcedir is a mountpoint, if requested. This should
		// reduce the risk of backing up an empty (unmounted) source on top of
		// a full destination.
		if b.config.SourceIsMountPoint {
			mounted, err := isMounted(b.config.SourceDir)
			if err != nil {
				return fmt.Errorf("Unable to verify if source_dir is mounted: %v", err)
			}
			if !mounted {
				return fmt.Errorf("SourceDir (%s) should be a mountpoint, but is not mounted", b.config.SourceDir)
			}
		}

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
		transp, err = transports.NewRcloneTransport(b.config, nil, b.log, b.dryRun)
	case "rdiff-backup":
		transp, err = transports.NewRdiffBackupTransport(b.config, nil, b.log, b.dryRun)
	case "rsync":
		transp, err = transports.NewRsyncTransport(b.config, nil, b.log, b.dryRun)
	default:
		return fmt.Errorf("Unknown transport %q", b.config.Transport)
	}
	if err != nil {
		return fmt.Errorf("Error creating %s transport: %v", b.config.Transport, err)
	}

	// Execute pre-commands, if any.
	if b.config.PreCommand != "" && !b.dryRun {
		if err := execute.Run("PRE", execute.WithShell(b.config.PreCommand), b.log); err != nil {
			return fmt.Errorf("Error running pre-command: %v", err)
		}
	}

	// Make it so...
	if err := transp.Run(); err != nil {
		return fmt.Errorf("Error running backup: %v", err)
	}

	// Execute post-commands, if any.
	if b.config.PostCommand != "" && !b.dryRun {
		if err := execute.Run("POST", execute.WithShell(b.config.PostCommand), b.log); err != nil {
			return fmt.Errorf("Error running post-command (possible backup failure): %v", err)
		}
	}

	return nil
}

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

// main
func main() {
	log = logger.New("")

	// Parse command line flags and read config file.
	if err := parseFlags(); err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	// Set verbose level
	verbose := int(opt.verbose)
	if verbose > 0 {
		log.SetVerboseLevel(verbose)
	}

	// Open and parse config file
	cfg, err := os.Open(opt.config)
	if err != nil {
		log.Fatalf("Unable to open config file: %v\n", err)
	}
	config, err := config.ParseConfig(cfg)
	if err != nil {
		log.Fatalf("Configuration error in %q: %v\n", opt.config, err)
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

	if opt.dryrun {
		log.Verboseln(1, "Warning: Dry-Run mode. Won't execute any commands.")
	}

	// Create new Backup and execute.
	b := NewBackup(config, log, opt.dryrun)

	if err = b.Run(); err != nil {
		log.Fatalln(err)
	}
	log.Verboseln(1, "*** Backup Result: Success")
}
