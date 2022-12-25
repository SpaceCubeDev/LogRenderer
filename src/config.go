package main

import (
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// serverTagRegexp can be used to check whether a server tag is valid or not
var serverTagRegexp = regexp.MustCompile(`[\w\-.]{1,64}`)

// SyntaxHighlightingConfig represents the syntax highlighting rules of a server
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
	pathPrefix  string
	// The regexps for the syntax highlighting for the logs of this server
	SyntaxHighlightingRegexps SyntaxHighlightingConfig `yaml:"syntax-highlighting"`
	// A pointer to the logs style dictionnary
	styles *map[string]string
}

type ClassicServerConfig struct {
	ServerConfig `yaml:",inline"` // saves lifes
	// The path of the log file to listen to
	LogFilePath string `yaml:"log-file-path"`
	// The path of the logs archive directory - only for classic servers
	ArchivedLogsDirPath string `yaml:"archived-logs-dir-path"`
	// The format of the archived log filenames - only for classic servers
	ArchivedLogFilenameFormat string `yaml:"archived-logs-filename-format"`
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

	// The path of the root dir, which must not include meta characters
	ArchivedLogsRootDir string `yaml:"archived-logs-root-dir"`
	// The pattern of the archived logs files
	ArchivedLogsFilePattern string `yaml:"archived-logs-file-pattern"`
	// Whether archive logs reading is enabled or not
	archivesEnabled bool
}

// Config represents the object version of the configuration file
type Config struct {
	// The port the web server will listen to
	Port uint16 `yaml:"port"`

	// The url prefix of the web interface (e.g. `/logrenderer` if the index of the application is `/logrenderer`). You leave this empty if the index is the root of your website (`/`)
	UrlPrefix string `yaml:"url-prefix"`

	// The url of the website home
	WebsiteHomeUrl string `yaml:"website-home-url"`
	// The url of the website logo
	WebsiteLogoUrl string `yaml:"website-logo-url"`
	// The url of the website favicon
	WebsiteFaviconUrl string `yaml:"website-favicon-url"`

	// Whether debug logs should be printed or not
	Debug bool `yaml:"debug"`

	// The delay before a new file watcher is started when a log file is reset/renamed
	DelayBeforeRewatch string `yaml:"delay-before-rewatch"`
	// The real value of DelayBeforeRewatch
	delayBeforeRewatch time.Duration

	// An optional prefix that will be added in front of each log file path,
	// for instance when the filesystem is mounted as a volume in a container at e.g. /mnt
	PathPrefix string `yaml:"path-prefix"`

	// The path of the file containing the logs style rules
	StyleFilePath string `yaml:"style-file-path"`
	// The styles as a map like name:css
	styles map[string]string

	// All the servers to list and listen to logs
	Servers struct {
		// The classic servers, whose log file path is static
		Classic []ClassicServerConfig `yaml:"classic"`
		// The dynamic servers, whose log file paths are potentially pointing to unprecise and several files
		Dynamic []DynamicServerConfig `yaml:"dynamic"`
	} `yaml:"servers"`
}

// getIdentifierFrom looks for the identifier in the given logFilePath using the server identifier regexp
func (servCfg *DynamicServerConfig) getIdentifierFrom(logFilePath string) (id string, found bool) {
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
	str += fmt.Sprintf("style-file-path: %s\n", config.StyleFilePath)
	str += "classic servers:\n"
	for _, servCfg := range config.Servers.Classic {
		str += "\t" + servCfg.ServerTag + ":\n"
		str += "\t\tdisplay-name: " + servCfg.DisplayName + "\n"
		str += "\t\tlog-file-path: " + servCfg.getLogFilePath() + "\n"
		if servCfg.archivesEnabled {
			str += "\t\tarchived-logs-dir-path: " + servCfg.getArchivedLogsDirPath() + "\n"
			str += "\t\tarchived-logs-filename-format: " + servCfg.ArchivedLogFilenameFormat + "\n"
		} else {
			str += "\t\tarchives not enabled\n"
		}
	}
	str += "dynamic servers:\n"
	for _, servCfg := range config.Servers.Dynamic {
		str += "\t" + servCfg.ServerTag + ":\n"
		str += "\t\tdisplay-name: " + servCfg.DisplayName + "\n"
		str += "\t\tlog-file-pattern: " + servCfg.getLogFilePattern() + "\n"
		str += "\t\tinstance-identifier: " + servCfg.InstanceIdentifier + "\n"
		if servCfg.archivesEnabled {
			str += "\t\tarchived-logs-root-dir: " + servCfg.getArchivedLogsRootDir() + "\n"
			str += "\t\tarchived-logs-file-pattern: " + servCfg.ArchivedLogsFilePattern + "\n"
		} else {
			str += "\t\tarchives not enabled\n"
		}
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
		return Config{}, fmt.Errorf("failed to parse delay-before-rewatch: %w", err)
	}
	if delay < 0 {
		return Config{}, errors.New("the delay-before-rewatch cannot be negative")
	}
	config.delayBeforeRewatch = delay

	config.styles, err = loadStyles(config.StyleFilePath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to load log styles file: %w", err)
	}

	if len(config.Servers.Classic) == 0 && len(config.Servers.Dynamic) == 0 {
		return Config{}, errors.New("no server found")
	}

	for servIndex := range config.Servers.Classic {
		servCfg := config.Servers.Classic[servIndex]
		servCfg.pathPrefix = config.PathPrefix
		err = servCfg.load(servIndex)
		if err != nil {
			return Config{}, err
		}
		servCfg.styles = &config.styles

		config.Servers.Classic[servIndex] = servCfg
	}
	for servIndex := range config.Servers.Dynamic {
		servCfg := config.Servers.Dynamic[servIndex]
		servCfg.pathPrefix = config.PathPrefix
		err = servCfg.load(servIndex)
		if err != nil {
			return Config{}, err
		}
		servCfg.styles = &config.styles

		config.Servers.Dynamic[servIndex] = servCfg
	}

	return config, nil
}

// loadStyles loads the logs styles declared in the file at the given path
func loadStyles(styleFilePath string) (map[string]string, error) {
	fileBytes, err := os.ReadFile(styleFilePath)
	if err != nil {
		return nil, err
	}

	var styles map[string]string
	err = yaml.Unmarshal(fileBytes, &styles)
	if err != nil {
		return nil, err
	}

	return styles, nil
}

func (servCfg *ClassicServerConfig) getLogFilePath() string {
	return filepath.Join(servCfg.pathPrefix, servCfg.LogFilePath)
}

func (servCfg *ClassicServerConfig) getArchivedLogsDirPath() string {
	return filepath.Join(servCfg.pathPrefix, servCfg.ArchivedLogsDirPath)
}

func (servCfg *ClassicServerConfig) load(servIndex int) error {
	err := checkFile(servCfg.getLogFilePath())
	if err != nil {
		return err
	}

	err = servCfg.loadCommon("classic", servIndex)
	if err != nil {
		return err
	}

	servCfg.archivesEnabled = servCfg.ArchivedLogsDirPath != ""
	if servCfg.archivesEnabled {
		err = checkDir(servCfg.getArchivedLogsDirPath())
		if err != nil {
			return err
		}
		if servCfg.ArchivedLogFilenameFormat == "" {
			return fmt.Errorf("no archive log filename format provided for classic server %q", servCfg.ServerTag)
		}
	}

	return nil
}

func (servCfg *DynamicServerConfig) getLogFilePattern() string {
	return filepath.Join(servCfg.pathPrefix, servCfg.LogFilePattern)
}

func (servCfg *DynamicServerConfig) getArchivedLogsRootDir() string {
	return filepath.Join(servCfg.pathPrefix, servCfg.ArchivedLogsRootDir)
}

func (servCfg *DynamicServerConfig) load(servIndex int) error {
	err := servCfg.loadCommon("dynamic", servIndex)
	if err != nil {
		return err
	}

	// LogFilePattern validity check
	if _, err = filepath.Match(servCfg.getLogFilePattern(), ""); err != nil {
		return fmt.Errorf("invalid log-file-pattern for dynamic server %q: %w", servCfg.ServerTag, err)
	}

	re, err := regexp.Compile(servCfg.InstanceIdentifier)
	if err != nil {
		return fmt.Errorf("invalid log-file-identifier regexp for dynamic server %q: %w", servCfg.ServerTag, err)
	}
	servCfg.logFileIdentifierRegexp = re

	servCfg.archivesEnabled = servCfg.ArchivedLogsRootDir != ""
	if servCfg.archivesEnabled {
		if servCfg.ArchivedLogsFilePattern == "" {
			return fmt.Errorf("no archived logs file pattern provided for dynamic server %q", servCfg.ServerTag)
		}
	}

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
		return fmt.Errorf("invalid server-tag for %s server %q: maximum length is 64 chars", servType, servCfg.ServerTag)
	case !serverTagRegexp.MatchString(servCfg.ServerTag):
		return fmt.Errorf("invalid server-tag for %s server %q: only alphanumerics chars, underscores and hypens are allowed", servType, servCfg.ServerTag)
	}

	if servCfg.DisplayName == "" {
		servCfg.DisplayName = servCfg.ServerTag
	}

	for i, regexField := range servCfg.SyntaxHighlightingRegexps {
		if regexField.Field == "" {
			printError(fmt.Errorf("invalid syntax highlighting field name for %s server %q, it will be ignored", servType, servCfg.ServerTag))
			servCfg.SyntaxHighlightingRegexps = append(servCfg.SyntaxHighlightingRegexps[:i], servCfg.SyntaxHighlightingRegexps[i+1:]...)
			continue
		}
		if regexField.Regex == "" {
			servCfg.SyntaxHighlightingRegexps[i].Regex = `/.^/` // Regexp that matches nothing
		}
	}

	return nil
}
