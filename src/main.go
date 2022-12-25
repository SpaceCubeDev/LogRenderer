package main

import (
	"errors"
	"flag"
	"fmt"
	"time"

	fifo "github.com/foize/go.fifo"
)

var version = "2.3.0-dev"

const instancesRefreshIntervalPerServer = 2 * time.Second

var doDebug bool

func main() {

	fmt.Print("\nStarting LogRenderer V"+version, " ...\n")

	configPath := flag.String("config", "./config.yml", "the path to the configuration file")
	flag.Parse()

	if *configPath == "" {
		exitWithError(errors.New("no configuration file path provided"))
	}

	config, err := loadConfigFrom(*configPath)
	if err != nil {
		exitWithError(err)
	}

	doDebug = config.Debug

	fmt.Print("Config:\n", config)

	outputChannel := make(chan Event, 16)
	hub := newHub()

	// classic servers startup
	for _, servCfg := range config.Servers.Classic {
		fmt.Println("Starting to watch for logs of classic server", servCfg.ServerTag, "...")

		hub.clientsByServer[servCfg.ServerTag] = []*Client{}

		logQueue := fifo.NewQueue()
		go watchServ(logQueue, watchProperties{
			servName:                  servCfg.ServerTag,
			logFilePath:               servCfg.getLogFilePath(),
			shouldRewatchOnFileRemove: true,
			delayBeforeRewatch:        config.delayBeforeRewatch,
		})
		go unstack(servCfg.ServerTag, logQueue, outputChannel)
	}
	// dynamic servers startup
	dynamicServers := make(DynamicServers)
	instancesRefreshInterval := instancesRefreshIntervalPerServer * time.Duration(len(config.Servers.Dynamic))
	for _, servCfg := range config.Servers.Dynamic {
		fmt.Println("Starting to watch for instances logs of dynamic server", servCfg.ServerTag, "...")

		server := newDynamicServer(servCfg)
		dynamicServers[servCfg.ServerTag] = server
		hub.clientsByDynamicServer[servCfg.ServerTag] = make(map[string][]*Client)

		go server.watchForInstances(hub, outputChannel, instancesRefreshInterval)
	}

	err = startServer(config, hub, outputChannel)
	if err != nil {
		exitWithError(err)
	}

	// hehehe
}
