package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"html/template"
	"os"
	"strconv"
	"time"
)

/*type SyntaxHighlightingConfig struct {
	Time  template.JS `yaml:"time"`
	Info  template.JS `yaml:"info"`
	Warn  template.JS `yaml:"warn"`
	Error template.JS `yaml:"error"`
	Text  template.JS `yaml:"text"`
}*/

type SyntaxHighlightingConfig []struct {
	Field template.JS `yaml:"field" json:"field"`
	Regex template.JS `yaml:"regex" json:"regex"`
}

// ServerConfig represents the properties of a server
type ServerConfig struct {
	// The raw name that will be used as an ID
	ServerTag string `yaml:"server-tag"`
	// The name that will be displayed on the web interface
	DisplayName string `yaml:"display-name"`
	// The path of the log file to listen to
	LogFilePath string `yaml:"log-file-path"`
	// The regexps for the syntax highlighting for the logs of this server
	SyntaxHighlightingRegexps SyntaxHighlightingConfig `yaml:"syntax-highlighting"`
}

// Config represents the object version of the configuration file
type Config struct {
	// The port the web server will listen to
	Port uint16 `yaml:"port"`

	// The url prefix of the web interface (e.g. `/logrenderer` if the index of the application is `/logrenderer`). You leave this empty if the index is the root of your website (`/`)
	UrlPrefix string `yaml:"url-prefix"`

	// The delay before a new file watcher is started when a log file is reset/renamed
	DelayBeforeRewatch string `yaml:"delay-before-rewatch"`
	// The real value of DelayBeforeRewatch
	delayBeforeRewatch time.Duration

	// All the servers to list and listen to logs
	Servers []ServerConfig `yaml:"servers"`
}

func (config Config) String() string {
	var str string
	str += fmt.Sprintf("port: %d\n", config.Port)
	str += fmt.Sprintf("url-prefix: %s\n", config.UrlPrefix)
	str += fmt.Sprintf("delay-before-rewatch: %s\n", config.delayBeforeRewatch)
	str += "servers:\n"
	for _, servCfg := range config.Servers {
		str += "\t" + servCfg.ServerTag + ":\n"
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

	delay, err := time.ParseDuration(config.DelayBeforeRewatch)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse delay-before-rewatch: %v", err)
	}
	if delay < 0 {
		return Config{}, errors.New("the delay-before-rewatch cannot be negative")
	}
	config.delayBeforeRewatch = delay

	if len(config.Servers) == 0 {
		return Config{}, errors.New("no server found")
	}

	for servIndex, servCfg := range config.Servers {
		err = checkFile(servCfg.LogFilePath)
		if err != nil {
			exitWithError(err)
		}

		if servCfg.DisplayName == "" {
			servCfg.DisplayName = servCfg.ServerTag
		}

		/*regexFields := reflect.ValueOf(&servCfg.SyntaxHighlightingRegexps)
		for i := 0; i < regexFields.Elem().NumField(); i++ {
			field := regexFields.Elem().Field(i)
			if field.String() == "" {
				field.SetString(`/.^/`)
			}
		}*/
		for i, regexField := range servCfg.SyntaxHighlightingRegexps {
			if regexField.Field == "" {
				printError(fmt.Errorf("invalid syntax highlighting field name for server %q, it will be ignored", servCfg.ServerTag))
				servCfg.SyntaxHighlightingRegexps = append(servCfg.SyntaxHighlightingRegexps[:i], servCfg.SyntaxHighlightingRegexps[i+1:]...)
				continue
			}
			if regexField.Regex == "" {
				servCfg.SyntaxHighlightingRegexps[i].Regex = `/.^/`
			}
		}

		config.Servers[servIndex] = servCfg
	}

	if config.UrlPrefix == "/" {
		config.UrlPrefix = ""
	}

	return config, nil
}
