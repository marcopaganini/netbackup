package config

// This file is part of netbackup (http://github.com/marcopaganini/netbackup)
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

// Main representation of a configuration file on disk.
type Config struct {
	Name        string
	SourceHost  string
	DestHost    string
	SourceDir   string
	DestDir     string
	ExtraArgs   string
	PreCommand  string
	PostCommand string
	Transport   string
	ExcludeList []string
	IncludeList []string
	Logfile     string
}

// Parse configuration file pointed by io.Reader and perform basic
// sanity checking. Returns a Config struct or error.
func ParseConfig(r io.Reader) (*Config, error) {
	config := &Config{}

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("Unable to read configuration file %v", err)
	}
	if json.Unmarshal(buf, config) != nil {
		return nil, fmt.Errorf("Error parsing configuration: %v", err)
	}

	// Basic sanity checking
	switch {
	case config.SourceDir == "":
		return nil, fmt.Errorf("SourceDir cannot be empty")
	case config.DestDir == "":
		return nil, fmt.Errorf("DestDir cannot be empty")
	case config.Name == "":
		return nil, fmt.Errorf("Name cannot be empty")
	case config.Transport == "":
		return nil, fmt.Errorf("Transport cannot be empty")
	}

	return config, nil
}
