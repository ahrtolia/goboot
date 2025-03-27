package main

import (
	"flag"
)

var configFile = flag.String("c", "config.yaml", "config file")

func main() {

	flag.Parse()

	app, err := CreateApp(*configFile)
	if err != nil {
		panic(err)
	}

	if err = app.Start(); err != nil {
		panic(err)
	}

	app.AwaitSignal()
}
