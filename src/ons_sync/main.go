package main

import (
	"os"
	"os/signal"
	"log"
	"flag"
)

func main() {
	addr := flag.String("addr", "198.13.60.39:8080", "REST API Server address")
	flag.Parse()
	log.SetFlags(0)

	DBConnect("198.13.60.39:28016", "ons_ledger", true)
	DBGetLatestUpdatedBlockInfo()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	onsEvtHandler, err := NewONSEventHandler(*addr, "/subscriptions")

	if err != nil {
		log.Printf("Failed to create ons event handler : ", err)
		os.Exit(2)
	}

	if onsEvtHandler.Run() == false {
		log.Printf("Failed to run ons event handler")
		os.Exit(2)
	}

	onsEvtHandler.Subscribe(true)

	sig := <-interrupt
	log.Println(sig)
	//interrupt가 발생하면.. (ctrl-c와 같은..)
	onsEvtHandler.Subscribe(false)
	onsEvtHandler.Terminate(true)
	DBDisconnect()
}