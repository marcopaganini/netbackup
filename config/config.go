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
)

// Config represents a configuration file on disk.  The fields in this struct
// *must* be tagged so we can correctly map them to the fields in the config
// file and detect extraneous configuration items.
type Config struct {
	Name        string   `ini:"name"`
	SourceHost  string   `ini:"source_host"`
	DestHost    string   `ini:"dest_host"`
	SourceDir   string   `ini:"source_dir"`
	DestDir     string   `ini:"dest_dir"`
	ExtraArgs   string   `ini:"extra_args"`
	PreCommand  string   `ini:"pre_command"`
	PostCommand string   `ini:"post_command"`
	Transport   string   `ini:"transport"`
	Exclude     []string `ini:"exclude"`
	Include     []string `ini:"include"`
	Logfile     string   `ini:"logfile"`
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

	// Basic sanity checking
	switch {
	case config.SourceDir == "":
		return nil, fmt.Errorf("source_dir cannot be empty")
	case config.DestDir == "":
		return nil, fmt.Errorf("dest_dir cannot be empty")
	case config.Name == "":
		return nil, fmt.Errorf("name cannot be empty")
	case config.Transport == "":
		return nil, fmt.Errorf("transport cannot be empty")
	}

	return config, nil
}
