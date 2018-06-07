package main

import (
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"os"
	ons "sawtooth_ons/ons_handler"
	"sawtooth_sdk/logging"
	"sawtooth_sdk/processor"
	"syscall"
)

var opts struct {
	Verbose []bool `short:"v" long:"verbose" description:"Increase verbosity"`
	Connect string `short:"C" long:"connect" description:"The validator component endpoint to" default:"tcp://localhost:4004"`
	PublicKey string `short:"p" long:"publickey" description:"ONS super user address" required:"true"`
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

	fmt.Printf("command line arguments: %v\n", os.Args)
	fmt.Printf("verbose = %v\n", len(opts.Verbose))
	fmt.Printf("endpoint = %v\n", opts.Connect)

	handler := &ons.ONSHandler{}

	//just for test yet.
	if handler.SetSudoAddress(opts.PublicKey) == false {
		logger.Debugf("Failed to set sudo address")
		os.Exit(2)
	}

	processor := processor.NewTransactionProcessor(opts.Connect)
	/*
		processor.SetMaxQueueSize(opts.Queue)
		if opts.Threads > 0 {
			processor.SetThreadCount(opts.Threads)
		}
	*/
	processor.AddHandler(handler)
	processor.ShutdownOnSignal(syscall.SIGINT, syscall.SIGTERM)
	err = processor.Start()
	if err != nil {
		logger.Error("Processor stopped: ", err)
	}
}
