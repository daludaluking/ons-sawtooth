package main

import (
	flags "github.com/jessevdk/go-flags"
	"os"
	"sawtooth_sdk/logging"
	//"sawtooth_sdk/processor"
	//"syscall"
	"fmt"
)

var opts struct {
	Verbose []bool `short:"v" long:"verbose" description:"Increase verbosity"`
	Connect string `short:"C" long:"connect" description:"The validator component endpoint to" default:"tcp://localhost:4004"`
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)

	logger := logging.Get()

	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			logger.Error("Failed to parse args: ", err)
			os.Exit(2)
		}
	}

	var loggingLevel int
	switch len(opts.Verbose) {
	case 0:
		loggingLevel = logging.WARN
	case 1:
		loggingLevel = logging.INFO
	default:
		loggingLevel = logging.DEBUG
	}
	logger.SetLevel(loggingLevel)

	logger.Debugf("command line arguments: %v", os.Args)
	logger.Debugf("verbose = %v\n", len(opts.Verbose))
	logger.Debugf("endpoint = %v\n", opts.Connect)

	//for debugging ... ...
	/*
	fmt.Printf("command line arguments: %v\n", os.Args)
	fmt.Printf("verbose = %v\n", len(opts.Verbose))
	fmt.Printf("endpoint = %v\n", opts.Connect)
	*/
}