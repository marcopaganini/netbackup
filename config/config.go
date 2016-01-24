package config

// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import (
	"fmt"
	"github.com/go-ini/ini"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
)

const (
	defaultLogDir = "/var/log/netbackup"
)

// Config represents a configuration file on disk.  The fields in this struct
// *must* be tagged so we can correctly map them to the fields in the config
// file and detect extraneous configuration items.
type Config struct {
	Name        string   `ini:"name"`
	SourceHost  string   `ini:"source_host"`
	DestHost    string   `ini:"dest_host"`
	DestDev     string   `ini:"dest_dev"`
	SourceDir   string   `ini:"source_dir"`
	DestDir     string   `ini:"dest_dir"`
	ExtraArgs   string   `ini:"extra_args"`
	FSCleanup   bool     `ini:"fs_cleanup"`
	PreCommand  string   `ini:"pre_command"`
	PostCommand string   `ini:"post_command"`
	Transport   string   `ini:"transport"`
	Exclude     []string `ini:"exclude" delim:" "`
	Include     []string `ini:"include" delim:" "`
	LogDir      string   `ini:"log_dir"`
	Logfile     string   `ini:"log_file"`
	// LUKS specific options
	LuksDestDev string `ini:"luks_dest_dev"`
	LuksKeyFile string `ini:"luks_keyfile"`
	// Rdiff-backup specific options
	RdiffBackupMaxAge int `ini:"rdiff_backup_max_age"`
}

// ParseConfig reads and parses a ini-style configuration from io.Reader and
// performs basic sanity checking on it. A pointer to Config is returned or
// error.
func ParseConfig(r io.Reader) (*Config, error) {
	config := &Config{}

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("Error reading config: %v", err)
	}

	inicfg, err := ini.Load(buf)
	if err != nil {
		return nil, fmt.Errorf("Error loading config: %v", err)
	}

	// For every key in the configuration file, make sure that a corresponding
	// tag exists in the tags for Config. This guarantees that typos in the
	// config file will generate an error.
	ref := reflect.TypeOf(*config)

	for _, inikey := range inicfg.Section("").KeyStrings() {
		found := false
		for ix := 0; ix < ref.NumField(); ix++ {
			skey := ref.Field(ix).Tag.Get("ini")
			if skey == inikey {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Unknown key %q in config file", inikey)
		}
	}

	if err := inicfg.MapTo(config); err != nil {
		return nil, fmt.Errorf("Error parsing config: %v", err)
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
	// Paths must be absolute if we're doing a local backup (no src or dst hosts.)
	case config.SourceHost == "" && !strings.HasPrefix(config.SourceDir, "/"):
		return nil, fmt.Errorf("source_dir must be an absolute path")
	case config.DestHost == "" && config.DestDir != "" && !strings.HasPrefix(config.DestDir, "/"):
		return nil, fmt.Errorf("dest_dir must be an absolute path")
	case config.DestDev != "" && !strings.HasPrefix(config.DestDev, "/"):
		return nil, fmt.Errorf("dest_dev must be an absolute path")
	case config.LuksDestDev != "" && !strings.HasPrefix(config.LuksDestDev, "/"):
		return nil, fmt.Errorf("dest_luks_dev must be an absolute path")
	// Specific checks
	case config.LuksDestDev != "" && config.LuksKeyFile == "":
		return nil, fmt.Errorf("dest_luks_dev requires luks_key_file")
	}

	return config, nil
}
