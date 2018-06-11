package main

import (
	"fmt"
	"syscall"
	"os"
	"os/user"
	"io/ioutil"
	"sawtooth_sdk/logging"
	"sawtooth_sdk/processor"
	ons "ons/ons_handler"
	flags "github.com/jessevdk/go-flags"
)

var opts struct {
	Verbose []bool `short:"v" long:"verbose" description:"Increase verbosity"`
	Connect string `short:"C" long:"connect" description:"The validator component endpoint to" default:"tcp://localhost:4004"`
	PublicKey string `short:"p" long:"publickey" description:"ONS super user address"`
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

	var local_public_key string
	if len(opts.PublicKey) == 0 {
		user, err := user.Current()
		path := user.HomeDir+"/.sawtooth/keys/"+user.Username+".pub"
		logger.Debugf("%s is used as public key\n", path)
		_public_key_bytes, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println("Fail to read private key.")
			os.Exit(2)
		}
		if _public_key_bytes[len(_public_key_bytes)-1] == 10 {
			_public_key_bytes = _public_key_bytes[:len(_public_key_bytes)-1]
		}
		local_public_key = string(_public_key_bytes)
		logger.Debugf("public key is %s\n", local_public_key)
	}else{
		local_public_key = opts.PublicKey
	}

	//for debugging ... ...
	fmt.Printf("command line arguments: %v\n", os.Args)
	fmt.Printf("verbose = %v\n", len(opts.Verbose))
	fmt.Printf("endpoint = %v\n", opts.Connect)

	handler := &ons.ONSHandler{}

	//just for test yet.
	if handler.SetSudoAddress(local_public_key) == false {
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
