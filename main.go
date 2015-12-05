package main

// netbackup - Consistent multi-method backup tool
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/marcopaganini/logger"
	"github.com/marcopaganini/netbackup/config"
	"github.com/marcopaganini/netbackup/transports"
)

var (
	// Generic logging object
	log *logger.Logger
)

// Transport lists the transport agent (rclone/rbackup/rdiff-backup)
// used to make a particular backup.
type Transport interface {
	Run() error
	SetLogFile(io.Writer) error
}

// Print error message and program usage to stderr, exit the program.
func usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
	}
	fmt.Fprintf(os.Stderr, "Usage%s:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	// Set verbose level
	log = logger.New("")

	// Parse command line flags and read config file.
	if err := parseFlags(); err != nil {
		log.Fatalf("Error: %v", err)
	}
	// Set verbose level
	if opt.verbose > 0 {
		log.SetVerboseLevel(int(opt.verbose))
	}
	if opt.dryrun {
		log.Verbosef(2, "Warning: Dry-Run mode. Won't execute any commands.")
	}

	cfg, err := os.Open(opt.config)
	if err != nil {
		log.Fatalf("Unable to open config file: %v", err)
	}

	//config, err := parseConfig(cfg)
	config, err := config.ParseConfig(cfg)
	if err != nil {
		log.Fatalf("Configuration error in %q: %v", opt.config, err)
	}

	if config.Transport == "rclone" {
		t, err := transports.NewRcloneTransport(config, nil, int(opt.verbose), opt.dryrun)
		if err != nil {
			log.Fatalf("Error creating rclone transport: %v", err)
		}
		if err := t.Run(); err != nil {
			log.Fatalln(err)
		}
	} else {
		log.Fatalf("Only rclone supported for now")
	}
}
