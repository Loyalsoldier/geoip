package main

import (
	"flag"
	"log"

	"github.com/Loyalsoldier/geoip/lib"
)

var (
	list       = flag.Bool("l", false, "List all available input and output formats")
	configFile = flag.String("c", "config.json", "URI of the JSON format config file, support both local file path and remote HTTP(S) URL")
)

func main() {
	flag.Parse()

	if *list {
		lib.ListInputConverter()
		lib.ListOutputConverter()
		return
	}

	instance, err := lib.NewInstance()
	if err != nil {
		log.Fatal(err)
	}

	if err := instance.Init(*configFile); err != nil {
		log.Fatal(err)
	}

	if err := instance.Run(); err != nil {
		log.Fatal(err)
	}
}
