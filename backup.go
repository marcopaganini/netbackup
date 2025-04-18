// This file is part of netbackup, a frontend to simplify periodic backups.
// For further information, check https://github.com/marcopaganini/netbackup
//
// (C) 2015-2024 by Marco Paganini <paganini AT paganini DOT net>

// main package.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/execute"
	"github.com/marcopaganini/netbackup/transports"
)

// Backup contains information for a given backup instance.
type Backup struct {
	config *config.Config
	dryRun bool
}

// NewBackup creates a new Backup instance.
func NewBackup(config *config.Config, dryRun bool) *Backup {
	// Create new Backup and execute.
	return &Backup{
		config: config,
		dryRun: opt.dryrun}
}

// mountDev mounts the destination device into a temporary mount point and
// returns the mount point name.
func (b *Backup) mountDev(ctx context.Context) (string, error) {
	tmpdir, err := os.MkdirTemp("", "netbackup_mount")
	if err != nil {
		return "", fmt.Errorf("unable to create temp directory: %v", err)
	}

	// We use the mount command instead of the mount syscall as it makes
	// simpler to specify defaults in /etc/fstab.
	cmd := []string{mountCmd, b.config.DestDev, tmpdir}
	if err := execute.Run(ctx, "MOUNT", cmd); err != nil {
		return "", err
	}

	return tmpdir, nil
}

// umountDev dismounts the destination device specified in config.DestDev.
func (b *Backup) umountDev(ctx context.Context) error {
	cmd := []string{umountCmd, b.config.DestDev}
	return execute.Run(ctx, "UMOUNT", cmd)
}

// openLuks opens the luks destination device into a temporary /dev/mapper
// device file and returns the /dev/mapper device filename.
func (b *Backup) openLuks(ctx context.Context) (string, error) {
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

	if err := execute.Run(ctx, "LUKS_OPEN", cmd); err != nil {
		return "", err
	}

	return devfile, nil
}

// closeLuks closes the current destination device.
func (b *Backup) closeLuks(ctx context.Context) error {
	// cryptsetup luksClose needs the /dev/mapper device name.
	cmd := []string{cryptSetupCmd, "luksClose", b.config.DestDev}
	return execute.Run(ctx, "LUKS_CLOSE", cmd)
}

// cleanFilesystem runs fsck to make sure the filesystem under config.dest_dev is
// intact, and sets the number of times to check to 0 and the last time
// checked to now. This option should only be used in EXTn filesystems or
// filesystems that support tunefs.
func (b *Backup) cleanFilesystem(ctx context.Context) error {
	// fsck (read-only check)
	cmd := []string{fsckCmd, "-n", b.config.DestDev}
	if err := execute.Run(ctx, "FS_CLEANUP", cmd); err != nil {
		return fmt.Errorf("error running %q: %v", cmd, err)
	}
	// Tunefs
	cmd = []string{tunefsCmd, "-C", "0", "-T", "now", b.config.DestDev}
	return execute.Run(ctx, "FS_CLEANUP", cmd)
}

// Run executes the backup according to the config file and options.
func (b *Backup) Run(ctx context.Context) error {
	var transp interface {
		Run(context.Context) error
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
				return fmt.Errorf("unable to verify if source_dir is mounted: %v", err)
			}
			if !mounted {
				return fmt.Errorf("source dir (%s) should be a mountpoint, but is not mounted", b.config.SourceDir)
			}
		}

		// Open LUKS device, if needed
		if b.config.LuksDestDev != "" {
			devfile, err := b.openLuks(ctx)
			if err != nil {
				return fmt.Errorf("error opening LUKS device %q: %v", b.config.LuksDestDev, err)
			}
			// Set the destination device to the /dev/mapper device opened by
			// LUKS. This should allow the natural processing to mount and
			// dismount this device.
			b.config.DestDev = devfile

			// close luks device at the end
			defer b.closeLuks(ctx)
			defer time.Sleep(2 * time.Second)
		}

		// Run cleanup on fs prior to backup, if requested.
		if b.config.FSCleanup {
			if err := b.cleanFilesystem(ctx); err != nil {
				return fmt.Errorf("error performing pre-backup cleanup on %q: %v", b.config.DestDev, err)
			}
		}

		// Mount destination device, if needed.
		if b.config.DestDev != "" {
			tmpdir, err := b.mountDev(ctx)
			if err != nil {
				return fmt.Errorf("error opening destination device %q: %v", b.config.DestDev, err)
			}
			// After we mount the destination device, we set Destdir to that location
			// so the backup will proceed seamlessly.
			b.config.DestDir = tmpdir

			// umount destination filesystem and remove temp mount point.
			defer os.Remove(b.config.DestDir)
			defer b.umountDev(ctx)
			// For some reason, not having a pause before attempting to unmount
			// can generate a race condition where umount complains that the fs
			// is busy (even though the transport is already down.)
			defer time.Sleep(2 * time.Second)
		}
	}

	var err error

	// Create new transport based on config.Transport
	switch b.config.Transport {
	case "custom":
		transp, err = transports.NewCustomTransport(b.config, nil, b.dryRun)
	case "rclone":
		transp, err = transports.NewRcloneTransport(b.config, nil, b.dryRun)
	case "rdiff-backup":
		transp, err = transports.NewRdiffBackupTransport(b.config, nil, b.dryRun)
	case "restic":
		transp, err = transports.NewResticTransport(b.config, nil, b.dryRun)
	case "rsync":
		transp, err = transports.NewRsyncTransport(b.config, nil, b.dryRun)
	default:
		return fmt.Errorf("unknown transport %q", b.config.Transport)
	}
	if err != nil {
		return fmt.Errorf("error creating %s transport: %v", b.config.Transport, err)
	}

	preCmdPresent := (b.config.PreCommand != "" && !b.dryRun)
	failCmdPresent := (b.config.FailCommand != "" && !b.dryRun)
	postCmdPresent := (b.config.PostCommand != "" && !b.dryRun)

	// Execute pre-commands, if any.
	if preCmdPresent {
		if err := execute.Run(ctx, "PRE-COMMAND", execute.WithShell(b.config.PreCommand)); err != nil {
			return fmt.Errorf("error running pre-command: %v", err)
		}
	}

	// Ignore interrupt signals and run the backup transport. If the user hits
	// Ctrl-C at this point (for example), both this process and the spawned
	// transport will receive SIGINT, and this will cause the transport to fail
	// and report error, but this program to be interrupted before it has a
	// chance to run FailCommand.
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM)
	err = transp.Run(ctx)
	signal.Reset(syscall.SIGINT, syscall.SIGTERM)

	// Execute post-commands if OK, or fail-command in case of failure.
	if err != nil {
		errbackup := err

		log.Verbosef(1, "Error running backup: %v\n", err)

		if failCmdPresent {
			log.Verbosef(1, "Running fail-command on backup error: %q\n", b.config.FailCommand)
			if err := execute.Run(ctx, "FAIL-COMMAND", execute.WithShell(b.config.FailCommand)); err != nil {
				log.Verbosef(1, "Error running fail-command: %v\n", err)
			}
		}
		return errbackup
	}

	// No errors.
	if postCmdPresent {
		if err := execute.Run(ctx, "POST-COMMAND", execute.WithShell(b.config.PostCommand)); err != nil {
			return fmt.Errorf("error running post-command (possible backup failure): %v", err)
		}
	}

	return nil
}
