package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var dynamicServerPathRegexp = regexp.MustCompile(`/dynamic/(?P<server>[\w\-.]{1,64})(/(?P<instance>[\w\-.]{1,64}))?`)

type ServerSummary struct {
	Tag, DisplayName string
	IsDynamic        bool
}

// CommonWebData contains data that can be accessed from anywhere and shared between the different pages
type CommonWebData struct {
	Version          string
	ExecDate         string
	UrlPrefix        string
	Servers          []ServerSummary
	MessageSeparator template.JS

	/* Archived logs related */
	AreArchivedLogsAvailable bool
	NoLogsLoadedYet          bool
	AvailableLogsArchives    []string
}

type handlerFunc func(w http.ResponseWriter, r *http.Request)

// var indexFunctionMap = template.FuncMap{"isServer": func() bool { return false }, "getCurrentServer": func() string { return "" }}

func getFuncMapFor(serv string, isIndex, isDynamic, isArchive bool) template.FuncMap {
	isServer := serv != ""
	return template.FuncMap{
		"isServer":         func() bool { return isServer },
		"getCurrentServer": func() string { return serv },
		"isIndex":          func() bool { return isIndex },
		"isDynamic":        func() bool { return isDynamic },
		"isArchive":        func() bool { return isArchive },
	}
}

func createLogHandlerFor(servCfg ClassicServerConfig, templateCommonData CommonWebData) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverHandler(w, r, templateCommonData, servCfg)
	}
}

func createArchiveHandlerFor(urlPrefix string, servCfg ClassicServerConfig, templateCommonData CommonWebData) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		parts = parts[2:] // get rid of the "archive" and server parts
		switch len(parts) {
		case 0:
			// list available logs
			listArchivesHandler(w, templateCommonData, servCfg)
		case 1:
			// display archived logs
			archiveHandler(w, r, parts[0], templateCommonData, servCfg)
		default:
			http.Redirect(w, r, urlPrefix+"/", http.StatusSeeOther)
		}
	}
}

func startServer(config Config, hub *Hub, outputChannel chan Event) error {
	go hub.run(outputChannel)

	serverNames := make([]ServerSummary, len(config.Servers.Classic)+len(config.Servers.Dynamic))
	templateCommonData := CommonWebData{
		Version:          "V" + version,
		UrlPrefix:        config.UrlPrefix,
		Servers:          serverNames,
		MessageSeparator: template.JS(messageSeparator),
	}

	var servIndex int
	// register a path for each server
	for _, servCfg := range config.Servers.Classic {
		serverNames[servIndex] = struct {
			Tag, DisplayName string
			IsDynamic        bool
		}{servCfg.ServerTag, servCfg.DisplayName, false}
		http.HandleFunc("/server/"+servCfg.ServerTag, createLogHandlerFor(servCfg, templateCommonData))
		http.HandleFunc("/archive/"+servCfg.ServerTag+"/", createArchiveHandlerFor(config.UrlPrefix, servCfg, templateCommonData))
		servIndex++
	}
	for _, servCfg := range config.Servers.Dynamic {
		serverNames[servIndex] = struct {
			Tag, DisplayName string
			IsDynamic        bool
		}{servCfg.ServerTag, strings.ReplaceAll(servCfg.DisplayName, "%id%", "<D>"), true}
		servIndex++
	}

	http.HandleFunc("/dynamic/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dynamic" || r.URL.Path == "/dynamic/" {
			dynamicServersListHandler(w, r, config.Servers.Dynamic) // sends back JSON
		} else {
			if serverTagRegexp.MatchString(r.URL.Path) {
				dynamicServerHandler(w, r, templateCommonData, config.Servers.Dynamic)
			} else {
				http.Redirect(w, r, config.UrlPrefix+"/", http.StatusSeeOther)
			}
		}
	})

	http.HandleFunc("/server", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, config.UrlPrefix+"/", http.StatusSeeOther)
	})

	http.HandleFunc("/archive", func(w http.ResponseWriter, r *http.Request) {
		// TODO: show the list of servers available for archive browsing
		http.Redirect(w, r, config.UrlPrefix+"/", http.StatusSeeOther)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
		} else {
			indexHandler(w, r, templateCommonData)
		}
	})

	http.HandleFunc("/ws", hub.serveWs)

	http.HandleFunc("/res/", serveResource)

	fmt.Println("Starting web server on", config.getWebServerAddress(), "...")

	return http.ListenAndServe(config.getWebServerAddress(), nil)
}

func indexHandler(w http.ResponseWriter, _ *http.Request, templateCommonData CommonWebData) {
	tmpl, err := parseTemplates([]string{"index", "navbar"}, getFuncMapFor("", true, false, false))
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	templateCommonData.ExecDate = time.Now().Format("15:04:05")
	err = tmpl.Execute(w, struct {
		CommonWebData
	}{
		CommonWebData: templateCommonData,
	})
	if err != nil {
		printError(err)
	}
}

func serverHandler(w http.ResponseWriter, r *http.Request, templateCommonData CommonWebData, servCfg ClassicServerConfig) {
	tmpl, err := parseTemplates([]string{"server", "navbar", "archive-loader"}, getFuncMapFor(servCfg.ServerTag, false, false, false))
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	availableLogFiles := []string{}
	if servCfg.archivesEnabled {
		availableLogFiles, err = listArchivedLogFiles(servCfg.ArchiveLogsDirPath, servCfg.ArchiveLogFilenameFormat)
		if err != nil {
			handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
			return
		}
	}

	maxLines := extractMaxLinesCount(r)

	templateCommonData.ExecDate = time.Now().Format("15:04:05")
	templateCommonData.AreArchivedLogsAvailable = true
	templateCommonData.NoLogsLoadedYet = false
	templateCommonData.AvailableLogsArchives = availableLogFiles
	err = tmpl.Execute(w, struct {
		CommonWebData
		Server                    string
		Instance                  string // just because the field is sometime used in the server template
		ServerDisplayName         string
		SyntaxHighlightingRegexps SyntaxHighlightingConfig
		ServerLogs                []string
	}{
		CommonWebData:             templateCommonData,
		Server:                    servCfg.ServerTag,
		ServerDisplayName:         servCfg.DisplayName,
		SyntaxHighlightingRegexps: servCfg.SyntaxHighlightingRegexps,
		ServerLogs:                getServerLogs(servCfg.LogFilePath, maxLines),
	})
	if err != nil {
		printError(err)
	}
}

func dynamicServersListHandler(w http.ResponseWriter, r *http.Request, dynamicServConfigs []DynamicServerConfig) {
	_ = r.ParseForm()
	only := r.FormValue("only")

	logFiles, status := getAllDynamicInstances(dynamicServConfigs, only)
	if status != http.StatusOK {
		prettier(w, "Internal error: please check the console", nil, int(status))
		return
	}

	if only != "" {
		if files, found := logFiles[only]; found {
			prettier(w, "Instances of dynamic server "+only, files, http.StatusOK)
		} else {
			prettier(w, "No instances found for dynamic server "+only, nil, http.StatusNotFound)
		}
	} else {
		prettier(w, "Instances found", logFiles, http.StatusOK)
	}
}

func dynamicServerHandler(w http.ResponseWriter, r *http.Request, templateCommonData CommonWebData, dynamicServConfigs []DynamicServerConfig) {
	namedGroups := findAllGroups(dynamicServerPathRegexp, r.URL.Path)
	serverTag := namedGroups["server"]
	serverId := namedGroups["instance"]
	if serverTag == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if serverId == "" {
		http.Redirect(w, r, "/dynamic?only="+serverTag, http.StatusSeeOther)
		return
	}

	servCfg, logFilePath, found := getDynamicServerConfigAndLogsPath(dynamicServConfigs, serverTag, serverId)
	if !found {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	tmpl, err := parseTemplates([]string{"server", "navbar", "archive-loader"}, getFuncMapFor(serverTag, false, true, false))
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	maxLines := extractMaxLinesCount(r)

	templateCommonData.ExecDate = time.Now().Format("15:04:05")
	templateCommonData.AreArchivedLogsAvailable = false
	templateCommonData.NoLogsLoadedYet = false
	err = tmpl.Execute(w, struct {
		CommonWebData
		Server                    string
		Instance                  string
		ServerDisplayName         string
		SyntaxHighlightingRegexps SyntaxHighlightingConfig
		ServerLogs                []string
	}{
		CommonWebData:             templateCommonData,
		Server:                    servCfg.ServerTag,
		Instance:                  serverId,
		ServerDisplayName:         strings.ReplaceAll(servCfg.DisplayName, "%id%", serverId),
		SyntaxHighlightingRegexps: servCfg.SyntaxHighlightingRegexps,
		ServerLogs:                getServerLogs(logFilePath, maxLines),
	})
	if err != nil {
		printError(err)
	}
}

func archiveHandler(w http.ResponseWriter, r *http.Request, logFile string, templateCommonData CommonWebData, servCfg ClassicServerConfig) {
	tmpl, err := parseTemplates([]string{"archive", "navbar", "archive-loader"}, getFuncMapFor(servCfg.ServerTag, false, false, true))
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	availableLogs, err := listArchivedLogFiles(servCfg.ArchiveLogsDirPath, servCfg.ArchiveLogFilenameFormat)
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	maxLines := extractMaxLinesCount(r)

	templateCommonData.ExecDate = time.Now().Format("15:04:05")
	templateCommonData.AreArchivedLogsAvailable = true
	templateCommonData.NoLogsLoadedYet = false
	templateCommonData.AvailableLogsArchives = availableLogs
	err = tmpl.Execute(w, struct {
		CommonWebData
		Server                    string
		ServerDisplayName         string
		SyntaxHighlightingRegexps SyntaxHighlightingConfig
		ServerLogs                []string
	}{
		CommonWebData:             templateCommonData,
		Server:                    servCfg.ServerTag,
		ServerDisplayName:         servCfg.DisplayName,
		SyntaxHighlightingRegexps: servCfg.SyntaxHighlightingRegexps,
		ServerLogs:                getArchiveLogs(filepath.Join(servCfg.ArchiveLogsDirPath, logFile), maxLines),
	})
	if err != nil {
		printError(err)
	}
}

func listArchivesHandler(w http.ResponseWriter, templateCommonData CommonWebData, servCfg ClassicServerConfig) {
	tmpl, err := parseTemplates([]string{"archive", "navbar", "archive-loader"}, getFuncMapFor(servCfg.ServerTag, false, false, true))
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	availableLogs, err := listArchivedLogFiles(servCfg.ArchiveLogsDirPath, servCfg.ArchiveLogFilenameFormat)
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	templateCommonData.ExecDate = time.Now().Format("15:04:05")
	templateCommonData.AreArchivedLogsAvailable = true
	templateCommonData.NoLogsLoadedYet = true
	templateCommonData.AvailableLogsArchives = availableLogs
	err = tmpl.Execute(w, struct {
		CommonWebData
		Server                    string
		ServerDisplayName         string
		SyntaxHighlightingRegexps SyntaxHighlightingConfig
	}{
		CommonWebData:             templateCommonData,
		Server:                    servCfg.ServerTag,
		ServerDisplayName:         servCfg.DisplayName,
		SyntaxHighlightingRegexps: servCfg.SyntaxHighlightingRegexps,
	})
	if err != nil {
		printError(err)
	}
}
