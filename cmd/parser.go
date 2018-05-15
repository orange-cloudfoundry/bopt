package cmd

import (
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

type Options struct {
	// Example of verbosity with level
	Verbose bool `short:"v" long:"verbose" description:"Verbose output"`
}

var options Options

var parser = flags.NewParser(&options, flags.HelpFlag|flags.PassDoubleDash)

var LogWriter io.Writer = os.Stderr

func Parse(args []string) error {
	log.SetOutput(LogWriter)
	_, err := parser.ParseArgs(args[1:])
	if err != nil {
		return err
	}
	if options.Verbose {
		log.SetLevel(log.DebugLevel)
	}
	return nil
}
