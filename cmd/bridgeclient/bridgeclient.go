package main

import (
	"flag"
	"log"
	"os"

	"eait.uq.edu.au/jscarsbrook/jsdata/v2/pkg/tsbridge"
)

var (
	filename = flag.String("filename", "", "The file to read and send to the backend.")
)

func main() {
	flag.Parse()

	bridge := tsbridge.NewBridge("")

	fileContents, err := os.ReadFile(*filename)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := bridge.Call(tsbridge.Request{
		Filename:     *filename,
		FileContents: string(fileContents),
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("resp = %+v", resp)
}
