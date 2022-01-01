package main

import (
	"errors"
	"flag"
	"fmt"
	fifo "github.com/foize/go.fifo"
)

const version = "1.1.8"

// func mainn(configPath *string) {

func main() {

	fmt.Print("\nStarting LogRenderer V"+version, " ...\n")

	configPath := flag.String("config", "", "the path to the configuration file")
	flag.Parse()

	if *configPath == "" {
		exitWithError(errors.New("no configuration file path provided"))
	}

	config, err := loadConfigFrom(*configPath)
	if err != nil {
		exitWithError(err)
	}

	fmt.Print("Config:\n", config)

	outputChannel := make(chan Event, 16)

	for _, servCfg := range config.Servers {
		fmt.Println("Starting to watch for logs of server", servCfg.ServerTag, "...")

		logQueue := fifo.NewQueue()
		go watchServ(servCfg.ServerTag, servCfg.LogFilePath, logQueue, config.delayBeforeRewatch)
		go unstack(servCfg.ServerTag, logQueue, outputChannel)
	}

	err = startServer(config, outputChannel)
	if err != nil {
		exitWithError(err)
	}

	// hehehe
}
