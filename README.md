# netbackup

![Code tests](https://github.com/marcopaganini/netbackup/workflows/Code%20Tests/badge.svg)

## Introduction

Netbackup is a CLI frontend for different backup programs (rsync, rdiff-backup, rclone). It simplifies the task of making local and remote backups by standardizing your multiple backup methods into a single program and configuration.

Netbackup eliminates the need for many different shell scripts with obscure options and ad-hoc commands, or long and unreadable invocations of rsync/rdiff-backup/rclone from your cron configuration. Most configuration options are read from config files, which could be kept under version control.  Netbackup also provides automatic logging of your backup results, with standardized log file names.

Despite the name, netbackup can (and should) be used for local backup operations as well. The "net" in the name is there for historical reasons; the idea was conceived a long time ago, purely for remote rsync backups but was since expanded to be a more generic program (the name stuck, though.) 

## Installation

For now, installation is simple: Just download the statically linked binary from github and copy it to a directory in your PATH. A good candidate is `/usr/local/bin`.

For those who prefer installing from source, clone the [Github netbackup repository](http://github.com/marcopaganini/netbackup) and type ``go build`` in the netbackup directory. You'll need golang installed to compile netbackup.

More convenient installation methods will come soon.

## A word about transports

The concept of a "transport" is central to the program's operation. Netbackup itself is only a front-end, and uses common backup programs to copy and verify data. The following transports are available.  

### rsync

Uses [rsync](https://rsync.samba.org/) to transport files locally or over the network. Rsync is a stable and fast method for backups and should be the primary choice, unless versioning or cloud backups are required.

### rdiff-backup

Uses [rdiff-backup](http://www.nongnu.org/rdiff-backup/) to copy files.  Rdiff-backup is a good alternative when versioning is required; it can keep the last N versions of a file around for later retrieval. Given its extra capabilities, rdiff-backup is generally fussier and slower than rsync.

### rclone

Uses [rclone](http://rclone.org/) to copy files. Rclone is centered on cloud operations and the transport of choice for cloud backups. It also works locally, but has fewer options than rsync or rdiff-backup.  

## Running netbackup

Most of the configuration of netbackup goes into a INI style configuration file, and a typical run of netbackup is something like:

```bash
$ netbackup --config=backup_config.conf
```

The configuration parser is based on [go-ini](http://github.com/go-ini/ini) and very flexible. Values can be specified with or without quotes.

Typing `netbackup` alone will show a short usage help. The options should be self-explanatory.

If you want to see what's going to be executed, use the `--dry-run` command-line option (or its abbreviated form, `-n`). This will show the command to be executed and the parsed include and exclude directives.

### Examples

This section contains a few examples of configuration files. Check the "Configuration reference" section for a more detailed description of each configuration directive.

#### Simple backup using rsync

This will copy all files from "/tmp" to "/backup" using rsync as the transport. It's one of the simplest possible backup configurations. Note that "name", "transport", "source_dir", and "dest_dir" are always required in a backup configuration file.

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

Not all files or directories need to be backed up:

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
dest_dir = "/backup"
dest_host = "remotehost"
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

Versioned backups are possible by using rdiff-backup as the transport (rdiff-backup must be installed separately.) It's also possible to specify how many days of versioning data should be kept at the destination with the `rdiff_backup_max_age` directive:

```
name = "vbackup"
transport = "rdiff-backup"
source_dir = "/data"
dest_dir = "/backup"
rdiff_backup_max_age = 10
```

This will keep ten days worth of backup versions (which can be recovered directly with rdiff-backup). It's also worth mentioning that rdiff-backup always keeps the files on disk at the latest version. The versions themselves are stored in the `rdiff-backup-data` directory at the destination.

#### Specifying arbitrary options to the transports

You can also pass arbitrary options to the transports. In the example, we tell rsync to skip files based on checksum, not timestamps:

```
name = "extrabackup"
transport = "rsync"
source_dir = "/data"
dest_dir = "/backup"
extra_args = "--checksum"
```

#### Backing up to the cloud

The rclone transport is an excellent choice to make daily backups of your irreplaceable data to a cloud provider. Naturally, rclone must be installed and properly configured (with `rclone --config`) before it can be used by netbackup. This is a somewhat complex example of backing up your photos to Google Drive (actually, a modified version of one of my config files):

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

* Copy all jpeg and XCF (Gimp) image files from all subdirectories under "/pics/family" and "/pics/scans" to the "Pictures" folder in Google Drive.
* The name "gdrive", used as the destination host above is actually the name given to your Google Drive source/destination when you configure rclone with `rclone --config`.
* Since a `include` directive exists, only those files will be synced. Check the [rclone documentation on filtering](http://rclone.org/filtering/) for further details.
* In this particular case, netbackup runs in a crontab as the root user, but the `rclone --config` was run as user "myuser". We need to specify the location of rclone's configuration file with `extra_args`.
* We also use `extra_args` to limit the bandwidth.

#### Backing up to an unmounted filesystem.

Some of my backup disks are permanently turned off and are only mounted when I need to make a backup. Netbackup knows how to mount a filesystem, clean it up, back up the data and then dismount it. Just specify `dest_dev` instead of `dest_dir`

```
name = "offline-backup"
transport = "rsync"
source_dir = "/data"
dest_dev = "/dev/disk/by-uuid/7aa76275-87f1-4baf-ae3c-7812481c2cb1"
fs_cleanup = "yes"
```

When backing up to a yet-to-be-mounted filesystem, it's a good idea to use the `fs_cleanup` option. When this option is present, netbackup will fsck the filesystem and set the fsck counters before mounting it. Also, the backup won't proceed if unrecoverable errors are found on the mount point.

**WARNING**: Given the almost unpredictable nature of device naming on modern versions of Linux, it's a good idea to use the UUID versions for the device names. Look at your "/dev/disk/by-uuid" directory to determine the correct device to use.

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

* `luks_dest_dev` must point to a device you'd normally open with "cryptsetup luksOpen". Netbackup will create a temporary "/dev/mapper" file and open this encrypted volume, pointing (internally) `backup_dev` to it.
* The `luks_keyfile` points to a file containing the key to open the luks device. The entire file content is considered as the key! If using an ASCII password, keep in mind that *newline characters count as part of the key*! (tip: use "echo -n your_password >file" to create a file without a newline at the end.)
* Netbackup will *not* create the encrypted volume; You must create it with cryptsetup and save the password into a file (I suggest a USB storage device for that.)

## Configuration Reference

Coming soon...


