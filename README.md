# netbackup

![Code tests](https://github.com/marcopaganini/netbackup/workflows/Code%20Tests/badge.svg)

## Introduction

Netbackup is a CLI front-end for different backup programs (rsync, rdiff-backup, rclone, and restic). It simplifies the task of making local and remote backups by standardizing multiple backup methods into a single program and configuration.

Netbackup eliminates the need for many different shell scripts with obscure options and ad-hoc commands, or long and unreadable invocations of rsync/rdiff-backup/rclone from your cron configuration. Most configuration options are read from config files, which could be kept under version control.  Netbackup also provides automatic logging of your backup results, with standardized log file names.

Despite its name, netbackup can (and should) be used for local backup operations as well. The "net" in the name is there for historical reasons as it started purely as a remote backup tool, but grew up to be a general purpose program.

## Installation

For now, installation is simple: Just download the statically linked binary from github and copy it to a directory in your path. A good candidate is `/usr/local/bin`.

Those who prefer to install from source, can easily do so using the commands below:

```bash
git clone http://github.com/marcopaganini/netbackup
sudo make install
```

You'll need the Go compiler and make installed to compile and install netbackup.

## A word about transports

The concept of a "transport" is central to the program's operation. Netbackup itself is only a front-end, and uses common backup programs to copy and verify data. The following transports are available.

### rsync

Uses [rsync](https://rsync.samba.org/) to transport files locally or over the network. Rsync is a stable and fast method for backups and should be the primary choice, unless versioning or cloud backups are required.

### rdiff-backup

Uses [rdiff-backup](http://www.nongnu.org/rdiff-backup/) to copy files.  Rdiff-backup is a good alternative when versioning is required; it can keep the last N versions of a file around for later retrieval. Given its extra capabilities, rdiff-backup is generally fussier and slower than rsync.

### rclone

Uses [rclone](http://rclone.org/) to copy files. This is the transport of choice for cloud backups. It also works locally, but has fewer options than rsync or rdiff-backup.

### restic

[Restic](https://restic.net) is a modern backup program that provides integration verification and encryption of your data. Works for local and remote backups.

## Running netbackup

Most of the configuration of netbackup goes into a ini style configuration file. Values can be specified with or without quotes. Options with multiple values work as a JSON array of strings (E.g.: `include=["/a", "/b"]`.

A typical run of netbackup is something like:

```bash
$ netbackup --config=backup_config.conf
```

The idea is to have multiple config files, one for each backup.

Typing `netbackup` alone will show a short usage help. The options should be self-explanatory.

To show the commands without actually executing them, use the `--dry-run` command-line option (or its abbreviated form, `-n`).

### Examples

This section contains a few examples of configuration files. Check the "Configuration reference" section for a more detailed description of each configuration directive.

#### Simple backup using rsync

Copy all files from "/tmp" to "/backup" using rsync as the transport. This is one of the simplest possible backup configurations. Note that "name", "transport", "source_dir", and "dest_dir" are always required in a backup configuration file.

```
# This is a comment. We all like comments.
# Everything starting with # is ignored
# inside the configuration files.

# All backups need a name
name = "example1"
transport = "rsync"
source_dir = "/tmp"
dest_dir = "/backup"
```

#### Excluding files

It's also possible to backup only a few directories under `source_dir`:

```
name = "anotherbackup"
transport = "rsync"
source_dir = "/tmp"
dest_dir = "/backup"
exclude = "/foo /bar"
```

Observe how we use "/foo" and "/bar" instead of "/tmp/foo" and "/tmp/bar" in the exclusion list. The reason is that rsync, rdiff-backup, and rclone all have different ways to specify file inclusion and exclusion (in this particular case, rsync uses the source directory as the "root" to match exclusions and inclusions).

To provide the highest degree of flexibility, the value of `include` and `exclude` are transport dependent. Netbackup will parse the value of the options in the config file as a string (using space as a delimiter), and create an appropriate file with the right contents for the transport to use. We may consider more sophisticated approaches in the future.

#### Backing up to a remote machine

To back up to a remote machine, just use the `dest_host` directive in your backup configuration. The example below shows how to back up the "/tmp" directory from the local host to the "/backup" directory at "remotehost". Note that netbackup assumes rsync can reach the destination machine (you must have passwordless SSH setup or any other method that rsync can use without requiring user interaction).

```
name = "backup_to_remote"
transport = "rsync"
source_dir = "/tmp"
dest_host = "remotehost"
dest_dir = "/backup"
```

#### Backing up from a remote host into the local machine

In a similar way, it's possible to backup from a remote host into the local machine with the `source_host` directive:

```
name = "central_backup"
transport = "rsync"
source_host = "sourcemachine"
source_dir = "/remote/dir"
dest_dir = "/local/dir"
```

#### Versioned backups

Versioned backups are possible by using `rdiff-backup` as the transport (rdiff-backup must be installed separately.) It's also possible to specify how many days of versioning data should be kept at the destination with the `expire_days` directive:

```
name = "vbackup"
transport = "rdiff-backup"
source_dir = "/data"
dest_dir = "/backup"
expire_days = 10
```

This will keep ten days worth of backup versions (which can be recovered directly with rdiff-backup). It's also worth mentioning that rdiff-backup always keeps the files on disk at the latest version. The past versions themselves are stored in the `rdiff-backup-data` directory at the destination.

#### Specifying arbitrary options to transports

You can also pass arbitrary options to the transports. In the example, we tell rsync to skip files based on checksum, not timestamps:

```
name = "extrabackup"
transport = "rsync"
source_dir = "/data"
dest_dir = "/backup"
extra_args = "--checksum"
```

#### Backing up to the cloud

The rclone transport is an excellent choice to make daily backups of your irreplaceable data to a cloud provider. Naturally, rclone must be installed and properly configured (with `rclone --config`) before it can be used by netbackup. This is a somewhat complex example of backing up your photos to Google Drive:

```
# Sync pictures to Google Drive, no raw images

name = "rclone-pictures"
transport = "rclone"
source_dir = "/pics"
dest_dir = "Pictures"
dest_host = "gdrive"

# JPG, GPX and MD5 files only
include="/family/**.{jpg,xcf} /scans/**.{jpg,xcf}"

# Specify the auth file and limit bandwidth
extra_args="--config=/home/myuser/.rclone.conf --bwlimit=512k"
```

What this does:

* Copy all jpeg and xcf (Gimp) image files from all subdirectories under "/pics/family" and "/pics/scans" to the "Pictures" folder in Google Drive.
* The name "gdrive", used as the destination host above is actually the name given to your Google Drive source/destination when you configure rclone with `rclone --config`.
* Since a `include` directive exists, only those files will be synced. Check the [rclone documentation on filtering](http://rclone.org/filtering/) for further details.
* In this particular case, netbackup runs in a crontab as the root user, but the `rclone --config` was executed as user "myuser". We need to specify the location of rclone's configuration file with `extra_args`.
* We also use `extra_args` to limit the bandwidth.

#### Backing up to an unmounted filesystem.

It's also possible to keep a destination disk unmounted and have netbackup mount it for backup (dismounting it at the end.) This provides an extra layer of protection against accidental erasure and allows us to "spin down" the disk when not in use, saving power and generating less noise. To do that, just provide a backup device with `dest_dev` instead of `dest_dir` in the example below:

```
name = "offline-backup"
transport = "rsync"
source_dir = "/data"
dest_dev = "/dev/disk/by-uuid/7aa76275-87f1-4baf-ae3c-7812481c2cb1"
fs_cleanup = "yes"
```

When backing up to a yet-to-be-mounted filesystem, it's a good idea to use the `fs_cleanup` option. When this option is present, netbackup will run `fsck` on the filesystem and set the fsck counters before mounting it. Also, the backup won't proceed if unrecoverable errors are found on the mount point.

**WARNING**: Given the almost unpredictable nature of device naming on modern versions of Linux, it's a good idea to use the UUID versions for the device names. Look at your `/dev/disk/by-uuid` directory to determine the correct device to use.

NOTE: Only extX filesystems are supported for now.

#### Backing up to an encrypted and unmounted filesystem.

It's always a good idea to encrypt your remote backups. Netbackup simplifies this task with the `luks_dest_dev` and `luks_keyfile` options:

```
name = "encrypted-backup"
transport = "rsync"
source_dir = "/data"
luks_dest_dev = "/dev/disk/by-uuid/e8607023-ef93-e7e5-914b-6af3e0430fb8"
luks_keyfile = "/media/foo/keyfile"
fs_cleanup = "yes"
```

Notes:

* `luks_dest_dev` must point to a device you'd normally open with `cryptsetup luksOpen`. Netbackup will create a temporary "/dev/mapper" file and open this encrypted volume, pointing (internally) `backup_dev` to it.
* The `luks_keyfile` points to a file containing the key to open the luks device. The entire file content is considered as the key! If using an ASCII password, keep in mind that *newline characters count as part of the key*! (tip: use `echo -n your_password >file` to create a file without a newline at the end.)
* Netbackup will *not* create the encrypted volume; You must create it with cryptsetup and save the password into a file (I suggest a USB storage device for that.)

## Configuration Reference

The list below contains all configuration parameters. Only a few parameters are mandatory and some are mutually exclusive. Netbackup will fail and issue an error if mutually exclusive options are present.

Parameters are listed in a somewhat logical order, with the most common and related options appearing first.

### name (string, mandatory)

This contains the name of the backup. Used to generate the full path of the log output.

### transport (string, mandatory)

The name of the transport (rsync, rclone, rdiff-backup, restic).

### source_host (string)

The name of the source host. Use this option to make backups of remote hosts into the current one. Netbackup uses SSH to reach the source host, if this option is present.

### dest_host (string)

The name of the destination host. If this option to make backups of the local host into remote hosts. Netbackup uses SSH to reach the destination host, if this option is present.

### source_dir (string, mandatory)

Source directory. Combine with `source_host` to copy from remote hosts into the local host.

### dest_dir / dest_dev (string, mandatory)

Destination directory *or* destination device for the backup. Either must be present, but not both.

Use `dest_dir` to specify a destination directory (must exist and be writable) or `dest_dev` to specify a destination device to use. If using a destination device, netbackup will automatically mount it as an extX filesystem and use it as the destination for the backup, unmounting it at the end.

### luks_dest_dev and luks_keyfile (string)

If `lust_dest_dev` is present on the configuration file, netbackup will attempt to open the device using `cryptsetup luksOpen` and mount it on a temporary mountpoint before the backup. This option normally requires `luks_keyfile`, which points to a keyfile containing the key used to open the LUKS device.

### source_is_mountpoint (boolean)

Fail the operation if the source is not a mounted filesystem. This option provides an extra level of safety against attempts to backup an empty directory source into an existing destination (which would cause netbackup to remove all data at the destination.)

### fs_cleanup (boolean)

Run `fsck` on the filesystem before the backup, and set the fsck count back to zero. This is mostly used with `dest_dev` to make sure the filesystem (which normally remains unmounted) is in a consistent state at the time of the backup. Use with extreme care. Supports extX only.

### expire_days (integer)

For transports that maintain history (rdiff-backup, restic) this specifies how far back (in days) we should keep history.

### extra_args (list of strings)

Add these arguments to the transport binary command-line. The value here does not replace the arguments generated by netbackup, but are added to the command-line *in addition* to them. There's no checking, so it is possible to create contradictory situations. Use with care.


### custom_bin (string)

Specify a custom name for the transport binary. One example would be to use a locally compiled version of your favorite transport. E.g: `custom_bin = rsync_beta`.

### pre_command (string)

Run this command (under the shell) *before* executing the backup. Will not proceed if the return code is not zero. Use this to perform any operations necessary before the backup starts. Terminate a chain of commands with `|| true` if you want them to never fail.

### post_command (string)

Run this command (under the shell) *after the backup finishes successfully*. Use this to unmount filesystems, notify operators, generate special snapshots or anything else that you need. Note that this only executes if the backup terminates successfully.

### fail_command (string)

Similar to `post_command` above, but only executes on backup failure.

### exclude and include (list of strings)

These options control which files to exclude and which files to include. They're transport dependent, so you should consult your selected transport for details. Excluded files are always listed first. For transports that support it (currently, rsync and rclone) the contents of these directives are converted into a "filter". This should be invisible to the user, but takes advantages of current best practices for these programs.

Here's a not so obvious example for rsync:

```
transport = "rsync"
source_dir = "/"
dest_dir = "/backup"

include = [
  "/etc/***",
  "/home/***",
  "/root/***",
  "/usr/local/bin/***",
  "/var/www/***",
]
exclude = [
  "*"
]
```

This will use rsync to *only* copy the contents of the directories above. Notice the use of "***" for rsync.
Without this, we'd need to explicitly include the full path to the last directory element. See the rsync(1) manpage for further details.

Another example using rclone:

```
transport = "rclone"
source_dir = "/data/foo"
dest_host = "google-drive"
dest_dir = "pvt"

exclude = [
  "/foobar/**"
]
```

If only the `exclude` directive is present, netbackup assumes "include everything else", so there's no need to add something like `include = [ "*" ]`. Note that the opposite is *not* true: When we only want to back up specific paths, the configuration must contain the `exclude = [ "*" ]` directive, or everything user `source_dir` will be copied (see the first example above).


### logdir (string)

The directory where netbackup will save the command output. The files are named `<name>/netbackup-<name>-YYYY-MM-DD.log` under this directory. The default value for `logdir` is `/var/log/netbackup`. Make sure the user running netbackup can write under this location.

### logfile (string)

Override the automatic filename generation and logging directory. Netbackup will send output directly into this file.

## Suggestions and bug reports

Feel free to open bug reports or suggest features in the [Issues](https://github.com/marcopaganini/netbackup/issues) page. PRs are always welcome, but please discuss your feature/bugfix first by creating an issue.
