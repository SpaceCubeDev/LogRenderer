package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"html/template"
	"os"
	"strconv"
)

type SyntaxHighlightingConfig struct {
	Time  template.JS `yaml:"time"`
	Info  template.JS `yaml:"info"`
	Warn  template.JS `yaml:"warn"`
	Error template.JS `yaml:"error"`
	Text  template.JS `yaml:"text"`
}

// ServerConfig represents the properties of a server
type ServerConfig struct {
	// The raw name that will be used as an ID
	server string
	// The name that will be displayed on the web interface
	DisplayName string `yaml:"display-name"`
	// The path of the log file to listen to
	LogFilePath string `yaml:"log-file-path"`
	// The regexps for the syntax highlighting for the logs of this server
	SyntaxHighlightingRegexps SyntaxHighlightingConfig `yaml:"syntax-highlighting"`
}

func (servConfig ServerConfig) clone() ServerConfig {
	return ServerConfig{
		server:      servConfig.server,
		DisplayName: servConfig.DisplayName,
		LogFilePath: servConfig.LogFilePath,
	}
}

// Config represents the object version of the configuration file
type Config struct {
	// The port the web server will listen to
	Port uint16 `yaml:"port"`

	// All the servers to list and listen to logs
	Servers map[string]ServerConfig `yaml:"servers"`
}

func (config Config) String() string {
	var str string
	str += fmt.Sprintf("port: %d\n", config.Port)
	str += "servers:\n"
	for serv, servCfg := range config.Servers {
		str += "\t" + serv + ":\n"
		str += "\t\tdisplay-name: " + servCfg.DisplayName + "\n"
		str += "\t\tlog-file-path: " + servCfg.LogFilePath + "\n"
	}
	return str
}

// getWebServerAddress returns the address the web server will listen to with the format expected by http.ListenAndServ
func (config Config) getWebServerAddress() string {
	return ":" + strconv.FormatUint(uint64(config.Port), 10)
}

// loadConfigFrom loads the config from the given file path and returns a Config object, or an error if one occurs
func loadConfigFrom(configPath string) (Config, error) {
	fileBytes, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(fileBytes, &config)
	if err != nil {
		return Config{}, err
	}

	if len(config.Servers) == 0 {
		return Config{}, errors.New("no server found")
	}

	for serv, servCfg := range config.Servers {
		err = checkFile(servCfg.LogFilePath)
		if err != nil {
			exitWithError(err)
		}
		servCfg.server = serv
		if servCfg.DisplayName == "" {
			servCfg.DisplayName = serv
		}
		config.Servers[serv] = servCfg
	}

	return config, nil
}
