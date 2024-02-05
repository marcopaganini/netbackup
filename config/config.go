// This file is part of netbackup, a frontend to simplify periodic backups.
// For further information, check https://github.com/marcopaganini/netbackup
//
// (C) 2015-2024 by Marco Paganini <paganini AT paganini DOT net>

package config

import (
	"fmt"
	"io"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	defaultLogDir = "/var/log/netbackup"
)

// Config represents a configuration file on disk.  The fields in this struct
// *must* be tagged so we can correctly map them to the fields in the config
// file and detect extraneous configuration items.
type Config struct {
	Name               string   `toml:"name"`
	SourceHost         string   `toml:"source_host"`
	DestHost           string   `toml:"dest_host"`
	DestDev            string   `toml:"dest_dev"`
	SourceDir          string   `toml:"source_dir"`
	DestDir            string   `toml:"dest_dir"`
	ExpireDays         int      `toml:"expire_days"`
	ExtraArgs          []string `toml:"extra_args" delim:" "`
	FSCleanup          bool     `toml:"fs_cleanup"`
	PreCommand         string   `toml:"pre_command"`
	SourceIsMountPoint bool     `toml:"source_is_mountpoint"`
	PostCommand        string   `toml:"post_command"`
	FailCommand        string   `toml:"fail_command"`
	Transport          string   `toml:"transport"`
	Exclude            []string `toml:"exclude" delim:" "`
	Include            []string `toml:"include" delim:" "`
	LogDir             string   `toml:"log_dir"`
	Logfile            string   `toml:"log_file"`
	CustomBin          string   `toml:"custom_bin"`
	PromTextFile       string   `toml:"prometheus_textfile"`
	// LUKS specific options
	LuksDestDev string `toml:"luks_dest_dev"`
	LuksKeyFile string `toml:"luks_keyfile"`
}

// ParseConfig reads and parses TOML configuration from io.Reader and performs
// basic sanity checking on it. A pointer to Config is returned or error.
func ParseConfig(r io.Reader) (*Config, error) {
	config := &Config{}

	mdata, err := toml.DecodeReader(r, config)
	if err != nil {
		return nil, fmt.Errorf("Error loading config: %v", err)
	}
	if len(mdata.Undecoded()) != 0 {
		keys := []string{}
		for _, v := range mdata.Undecoded() {
			strv := v.String()
			keys = append(keys, strv)
		}
		return nil, fmt.Errorf("unknown field(s) in config: %s", strings.Join(keys, ","))
	}

	// Set defaults
	if config.Logfile == "" && config.LogDir == "" {
		config.LogDir = defaultLogDir
	}

	// Count the number of destinations set
	ndest := 0
	ndev := 0
	if config.DestDir != "" {
		ndest++
	}
	if config.DestDev != "" {
		ndev++
	}
	if config.LuksDestDev != "" {
		ndev++
	}

	// Basic config validation
	switch {
	// Base checks
	case config.Name == "":
		return nil, fmt.Errorf("name cannot be empty")
	case config.SourceDir == "":
		return nil, fmt.Errorf("source_dir cannot be empty")
	case config.Transport == "":
		return nil, fmt.Errorf("transport cannot be empty")
	case config.Logfile != "" && config.LogDir != "":
		return nil, fmt.Errorf("either log_dir or log_file can be set")
	// Make sure destination combos are valid.
	case (ndest + ndev) == 0:
		return nil, fmt.Errorf("no destination set")
	case (ndest + ndev) != 1:
		return nil, fmt.Errorf("only one destination (dest_dir, dest_dev, or luks_dest_dev) may be set")
	case ndev != 0 && config.DestHost != "":
		return nil, fmt.Errorf("cannot have dest_dev and dest_host set. Remote mounting not supported")
	case ndev == 0 && config.FSCleanup:
		return nil, fmt.Errorf("fs_cleanup can only be used when destination is a filesystem")
	// We can only check if source is a mount point for local backups.
	case config.SourceHost != "" && config.SourceIsMountPoint:
		return nil, fmt.Errorf("Cannot validate if source is a mountpoint with remote backups")
	// Paths must be absolute if we're doing a local backup (no src or dst hosts.)
	case config.SourceHost == "" && !strings.HasPrefix(config.SourceDir, "/"):
		return nil, fmt.Errorf("source_dir must be an absolute path")
	case config.DestHost == "" && config.DestDir != "" && !strings.HasPrefix(config.DestDir, "/"):
		return nil, fmt.Errorf("dest_dir must be an absolute path")
	case config.DestDev != "" && !strings.HasPrefix(config.DestDev, "/"):
		return nil, fmt.Errorf("dest_dev must be an absolute path")
	case config.LuksDestDev != "" && !strings.HasPrefix(config.LuksDestDev, "/"):
		return nil, fmt.Errorf("dest_luks_dev must be an absolute path")
	// Specific checks.
	case config.LuksDestDev != "" && config.LuksKeyFile == "":
		return nil, fmt.Errorf("dest_luks_dev requires luks_key_file")
	}

	return config, nil
}
