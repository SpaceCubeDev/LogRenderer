package main

import (
	"fmt"
	fifo "github.com/foize/go.fifo"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type DynamicServerInstance struct {
	id          string
	displayName string
	logFilePath string
	ended       bool
}

type DynamicServer struct {
	config    DynamicServerConfig
	tag       string // shorthand for config.ServerTag
	instances []*DynamicServerInstance
}

func (server DynamicServer) watchForInstances(hub *Hub, outputChannel chan Event, watchInterval time.Duration) {
	for {
		startTime := time.Now()

		stillRunning := make(map[string]struct{})
		for i := 0; i < len(server.instances); i++ {
			instance := server.instances[i]
			if instance.ended {
				server.instances = append(server.instances[:i], server.instances[i+1:]...)
				i-- // so as not to skip the next one
			} else {
				stillRunning[instance.id] = struct{}{}
			}
		}

		latestInstances, err := getDynamicServerInstances(server.config)
		if err != nil {
			printError(fmt.Errorf("failed to find instances of server %q: %v", server.tag, err))
		} else {
			for _, instance := range latestInstances {
				if _, exists := stillRunning[instance.id]; exists {
					continue // already watching it
				}
				debugPrint(fmt.Sprintf("Found new instance of server %q: %q", server.tag, instance.id))
				// preserve existing WS connections between instance reboots
				if _, exists := hub.clientsByDynamicServer[server.tag][instance.id]; exists {
					hub.sendResetMessage(server.tag, instance.id)
				} else {
					hub.clientsByDynamicServer[server.tag][instance.id] = []*Client{}
				}
				// instance given as parameter to not be replaced by the for loop current instance
				go func(instance *DynamicServerInstance) {
					logQueue := fifo.NewQueue()
					stop := false
					go unstackDynamic(server.tag, instance.id, logQueue, outputChannel, &stop)
					watchServ(logQueue, watchProperties{
						servName:                  joinWSServer(server.tag, instance.id),
						logFilePath:               instance.logFilePath,
						shouldRewatchOnFileRemove: false,
					})
					// watches until it returns
					stop = true
					instance.ended = true
				}(&instance)
				server.instances = append(server.instances, &instance)
			}
		}

		time.Sleep(watchInterval - time.Since(startTime))
	}
}

func newDynamicServer(config DynamicServerConfig) *DynamicServer {
	return &DynamicServer{config: config, tag: config.ServerTag, instances: []*DynamicServerInstance{}}
}

type DynamicServers map[string]*DynamicServer

func parseWSServer(raw string) (server, instance string, valid bool) {
	parts := strings.Split(raw, "=>")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func joinWSServer(server, instance string) string {
	return server + "=>" + instance
}

func getDynamicServerInstances(servCfg DynamicServerConfig) ([]DynamicServerInstance, error) {
	logFilesPaths, err := filepath.Glob(servCfg.LogFilePattern)
	if err != nil {
		return nil, fmt.Errorf("invalid log file pattern: %v", err)
	}
	logFiles := make([]DynamicServerInstance, len(logFilesPaths))
	for i, logFilePath := range logFilesPaths {
		id, found := servCfg.getIdentifierFrom(logFilePath)
		if !found {
			debugPrint(fmt.Sprintf("Instance identifier not found for server %q (%q) !", servCfg.ServerTag, logFilePath))
			continue
		}
		logFiles[i] = DynamicServerInstance{
			id: id,
			// infers the instance identifier into the server display name
			displayName: strings.ReplaceAll(servCfg.DisplayName, "%id%", id),
			logFilePath: logFilePath,
			ended:       false,
		}
	}
	return logFiles, nil
}

func getAllDynamicInstances(dynamicServConfigs []DynamicServerConfig, onlyThisServer string) (logFiles map[string]map[string]string, status uint) {
	logFiles = make(map[string]map[string]string)
	for _, servCfg := range dynamicServConfigs {
		if onlyThisServer != "" && servCfg.ServerTag != onlyThisServer {
			continue
		}
		logFilesPaths, err := filepath.Glob(servCfg.LogFilePattern)
		if err != nil {
			debugPrint(fmt.Sprintf("invalid log file pattern for server %q: %v", servCfg.ServerTag, err))
			return nil, http.StatusInternalServerError
		}
		logFiles[servCfg.ServerTag] = make(map[string]string)
		for _, logFilePath := range logFilesPaths {
			id, found := servCfg.getIdentifierFrom(logFilePath)
			if !found {
				debugPrint(fmt.Sprintf("Instance identifier not found for server %q (%q) !", servCfg.ServerTag, logFilePath))
				continue
			}
			// infers the instance identifier into the server display name
			logFiles[servCfg.ServerTag][id] = strings.ReplaceAll(servCfg.DisplayName, "%id%", id)
		}
	}
	return logFiles, http.StatusOK
}

func getDynamicServerConfigAndLogsPath(dynamicServConfigs []DynamicServerConfig, server string, instance string) (config DynamicServerConfig, logFilePath string, found bool) {
	for _, servCfg := range dynamicServConfigs {
		if servCfg.ServerTag != server {
			continue
		}
		logFilesPaths, err := filepath.Glob(servCfg.LogFilePattern)
		if err != nil {
			debugPrint(fmt.Sprintf("invalid log file pattern for server %q: %v", servCfg.ServerTag, err))
			return DynamicServerConfig{}, "", false
		}
		for _, logFilePath := range logFilesPaths {
			id, found := servCfg.getIdentifierFrom(logFilePath)
			if !found {
				debugPrint(fmt.Sprintf("Instance identifier not found for server %q !", servCfg.ServerTag))
				continue
			}
			if id == instance {
				return servCfg, logFilePath, true
			}
		}
	}
	return DynamicServerConfig{}, "", false
}
