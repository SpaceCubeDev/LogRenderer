package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
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

var serverTagRegexp = regexp.MustCompile(`[\w\-.]{1,64}`)

type SyntaxHighlightingConfig []struct {
	Field template.JS `yaml:"field" json:"field"`
	Regex template.JS `yaml:"regex" json:"regex"`
}

// ServerConfig represents the properties of a server
type ServerConfig struct {
	// The raw name that will be used as an ID
	ServerTag string `yaml:"server-tag"`
	// The name that will be displayed on the web interface.
	// For dynamic servers, it can contain a %id% placeholder that will be replaced by the instance identifier
	DisplayName string `yaml:"display-name"`
	// The regexps for the syntax highlighting for the logs of this server
	SyntaxHighlightingRegexps SyntaxHighlightingConfig `yaml:"syntax-highlighting"`
}

type ClassicServerConfig struct {
	ServerConfig `yaml:",inline"` // saves lifes
	// The path of the log file to listen to
	LogFilePath string `yaml:"log-file-path"`
	// The path of the logs archive directory - only for classic servers
	ArchiveLogsDirPath string `yaml:"archived-logs-dir-path"`
	// The format of the archived log filenames - only for classic servers
	ArchiveLogFilenameFormat string `yaml:"archive-log-filename-format"`
	// Whether archive logs reading is enabled or not
	archivesEnabled bool
}

type DynamicServerConfig struct {
	ServerConfig `yaml:",inline"` // saves lifes
	// The pattern of the log files to listen to
	LogFilePattern string `yaml:"log-file-pattern"`
	// A regexp that will be applied to LogFilePath to retrieve the identifier of the instance,
	// because there may be many instances under one dynamic server tag.
	// e.g: ~/servers/Lobby-(?P<id>\d*)_*/logs/latest.log, where id is the name of the group that contains the instance's identifier
	InstanceIdentifier string `yaml:"instance-identifier"`
	// The compiled regexp of the log file identifier
	logFileIdentifierRegexp *regexp.Regexp
}

// Config represents the object version of the configuration file
type Config struct {
	// The port the web server will listen to
	Port uint16 `yaml:"port"`

	// The url prefix of the web interface (e.g. `/logrenderer` if the index of the application is `/logrenderer`). You leave this empty if the index is the root of your website (`/`)
	UrlPrefix string `yaml:"url-prefix"`

	// Whether debug logs should be printed or not
	Debug bool `yaml:"debug"`

	// The delay before a new file watcher is started when a log file is reset/renamed
	DelayBeforeRewatch string `yaml:"delay-before-rewatch"`
	// The real value of DelayBeforeRewatch
	delayBeforeRewatch time.Duration

	// All the servers to list and listen to logs
	Servers struct {
		// The classic servers, whose log file path is static
		Classic []ClassicServerConfig `yaml:"classic"`
		// The dynamic servers, whose log file paths are potentially pointing to unprecise and several files
		Dynamic []DynamicServerConfig `yaml:"dynamic"`
	} `yaml:"servers"`
}

func (servCfg DynamicServerConfig) getIdentifierFrom(logFilePath string) (id string, found bool) {
	id = findAllGroups(servCfg.logFileIdentifierRegexp, logFilePath)["id"]
	return id, id != ""
}

// String returns a string representation of the Config
func (config Config) String() string {
	var str string
	str += fmt.Sprintf("port: %d\n", config.Port)
	str += fmt.Sprintf("url-prefix: %s\n", config.UrlPrefix)
	str += fmt.Sprintf("debug: %t\n", config.Debug)
	str += fmt.Sprintf("delay-before-rewatch: %s\n", config.delayBeforeRewatch)
	str += "classic servers:\n"
	for _, servCfg := range config.Servers.Classic {
		str += "\t" + servCfg.ServerTag + ":\n"
		str += "\t\tdisplay-name: " + servCfg.DisplayName + "\n"
		str += "\t\tlog-file-path: " + servCfg.LogFilePath + "\n"
		str += "\t\tarchived-logs-dir-path: " + servCfg.ArchiveLogsDirPath + "\n"
		str += "\t\tarchived-log-filename-format: " + servCfg.ArchiveLogsDirPath + "\n"
	}
	str += "dynamic servers:\n"
	for _, servCfg := range config.Servers.Dynamic {
		str += "\t" + servCfg.ServerTag + ":\n"
		str += "\t\tdisplay-name: " + servCfg.DisplayName + "\n"
		str += "\t\tlog-file-pattern: " + servCfg.LogFilePattern + "\n"
		str += "\t\tinstance-identifier: " + servCfg.InstanceIdentifier + "\n"
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

	if config.UrlPrefix == "/" {
		config.UrlPrefix = ""
	}

	delay, err := time.ParseDuration(config.DelayBeforeRewatch)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse delay-before-rewatch: %v", err)
	}
	if delay < 0 {
		return Config{}, errors.New("the delay-before-rewatch cannot be negative")
	}
	config.delayBeforeRewatch = delay

	if len(config.Servers.Classic) == 0 && len(config.Servers.Dynamic) == 0 {
		return Config{}, errors.New("no server found")
	}

	for servIndex, servCfg := range config.Servers.Classic {
		err = servCfg.load(servIndex)
		if err != nil {
			return Config{}, err
		}

		config.Servers.Classic[servIndex] = servCfg
	}
	for servIndex, servCfg := range config.Servers.Dynamic {
		err = servCfg.load(servIndex)
		if err != nil {
			return Config{}, err
		}

		config.Servers.Dynamic[servIndex] = servCfg
	}

	return config, nil
}

func (servCfg *ClassicServerConfig) load(servIndex int) error {
	err := checkFile(servCfg.LogFilePath)
	if err != nil {
		return err
	}

	err = servCfg.loadCommon("classic", servIndex)
	if err != nil {
		return err
	}

	servCfg.archivesEnabled = servCfg.ArchiveLogsDirPath != ""
	if servCfg.archivesEnabled {
		err = checkDir(servCfg.ArchiveLogsDirPath)
		if err != nil {
			return err
		}
		if servCfg.ArchiveLogFilenameFormat == "" {
			return fmt.Errorf("no archive log filename format provided for server %q", servCfg.ServerTag)
		}
	}

	return nil
}

func (servCfg *DynamicServerConfig) load(servIndex int) error {
	err := servCfg.loadCommon("dynamic", servIndex)
	if err != nil {
		return err
	}

	// LogFilePattern validity check
	if _, err = filepath.Match(servCfg.LogFilePattern, ""); err != nil {
		return fmt.Errorf("invalid log-file-pattern for server %q: %w", servCfg.ServerTag, err)
	}

	re, err := regexp.Compile(servCfg.InstanceIdentifier)
	if err != nil {
		return fmt.Errorf("invalid log-file-identifier regexp for server %q: %w", servCfg.ServerTag, err)
	}
	servCfg.logFileIdentifierRegexp = re

	return nil
}

// loadCommon verifies and adapt the values of the ServerConfig, while returning any fatal error.
// The servIndex param is used to identify the server in case the server-tag property is not defined
func (servCfg *ServerConfig) loadCommon(servType string, servIndex int) error {
	// ServerTag validation
	switch {
	case len(servCfg.ServerTag) == 0:
		return fmt.Errorf("no server-tag provided for registered %s server n°%d", servType, servIndex+1)
	case len(servCfg.ServerTag) > 64:
		return fmt.Errorf("invalid server-tag for server %q: maximum length is 64 chars", servCfg.ServerTag)
	case !serverTagRegexp.MatchString(servCfg.ServerTag):
		return fmt.Errorf("invalid server-tag for server %q: only alphanumerics chars, underscores and hypens are allowed", servCfg.ServerTag)
	}

	if servCfg.DisplayName == "" {
		servCfg.DisplayName = servCfg.ServerTag
	}

	for i, regexField := range servCfg.SyntaxHighlightingRegexps {
		if regexField.Field == "" {
			printError(fmt.Errorf("invalid syntax highlighting field name for server %q, it will be ignored", servCfg.ServerTag))
			servCfg.SyntaxHighlightingRegexps = append(servCfg.SyntaxHighlightingRegexps[:i], servCfg.SyntaxHighlightingRegexps[i+1:]...)
			continue
		}
		if regexField.Regex == "" {
			servCfg.SyntaxHighlightingRegexps[i].Regex = `/.^/` // Regexp that matches nothing
		}
	}

	return nil
}
