package main

import (
	"flag"
	"os"
	"strings"
)

type stringFlag struct {
	value string
	set   bool
}

func (s *stringFlag) String() string {
	return s.value
}

func (s *stringFlag) Set(val string) error {
	s.value = val
	s.set = true
	return nil
}

func main() {

	configFlag := &stringFlag{value: "config.yaml"}
	flag.Var(configFlag, "c", "config file")
	flag.Parse()

	configFile := configFlag.value
	if !configFlag.set {
		if envConfig := strings.TrimSpace(os.Getenv("CONFIG_NAME")); envConfig != "" {
			configFile = envConfig
		}
	}

	app, err := CreateApp(configFile)
	if err != nil {
		panic(err)
	}

	if err = app.Start(); err != nil {
		panic(err)
	}
}
