package main

import (
	"errors"
	"flag"
	"fmt"
	fifo "github.com/foize/go.fifo"
)

// func mainn(configPath *string) {

func main() {

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

	for serv, servCfg := range config.Servers {
		fmt.Println("Starting to watch for logs of server", serv, "...")

		logQueue := fifo.NewQueue()
		go watchServ(servCfg.LogFilePath, logQueue)
		go unstack(serv, logQueue, outputChannel)
	}

	err = startServer(config.getWebServerAddress(), config.Servers, outputChannel)
	if err != nil {
		exitWithError(err)
	}

	// hehehe
}
