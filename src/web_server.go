package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// CommonWebData contains data that can be accessed from anywhere
type CommonWebData struct {
	Version   string
	ExecDate  string
	UrlPrefix string
	Servers   []struct {
		Tag, DisplayName string
	}
	MessageSeparator template.JS

	/* Archived logs related */
	NoLogsLoadedYet       bool
	AvailableLogsArchives []string
}

type handlerFunc func(w http.ResponseWriter, r *http.Request)

// var indexFunctionMap = template.FuncMap{"isServer": func() bool { return false }, "getCurrentServer": func() string { return "" }}

func getFuncMapFor(serv string, isArchive bool) template.FuncMap {
	isServer := serv != ""
	return template.FuncMap{"isServer": func() bool { return isServer }, "getCurrentServer": func() string { return serv }, "isArchive": func() bool { return isArchive }}
}

func createLogHandlerFor(servCfg ServerConfig, templateCommonData CommonWebData) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverHandler(w, r, templateCommonData, servCfg)
	}
}

func createArchiveHandlerFor(urlPrefix string, servCfg ServerConfig, templateCommonData CommonWebData) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		parts = parts[2:] // get rid of the "archive" and server parts
		switch len(parts) {
		case 0:
			// list available logs
			listArchivesHandler(w, templateCommonData, servCfg)
		case 1:
			// display archived logs
			archiveHandler(w, parts[0], templateCommonData, servCfg)
		default:
			http.Redirect(w, r, urlPrefix+"/", http.StatusSeeOther)
		}
	}
}

func startServer(config Config, outputChannel chan Event) error {
	hub := newHub()
	go hub.run(outputChannel)

	serverNames := make([]struct{ Tag, DisplayName string }, len(config.Servers))
	templateCommonData := CommonWebData{
		Version:          "V" + version,
		UrlPrefix:        config.UrlPrefix,
		Servers:          serverNames,
		MessageSeparator: template.JS(messageSeparator),
	}

	// register a path for each server
	for i, servCfg := range config.Servers {
		serverNames[i] = struct{ Tag, DisplayName string }{Tag: servCfg.ServerTag, DisplayName: servCfg.DisplayName}
		hub.clientsByServer[servCfg.ServerTag] = []*Client{}
		http.HandleFunc("/server/"+servCfg.ServerTag, createLogHandlerFor(servCfg, templateCommonData))
		http.HandleFunc("/archive/"+servCfg.ServerTag+"/", createArchiveHandlerFor(config.UrlPrefix, servCfg, templateCommonData))
	}

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
	/*time.Sleep(5 * time.Second)
	return nil*/
}

func indexHandler(w http.ResponseWriter, r *http.Request, templateCommonData CommonWebData) {
	tmpl, err := parseTemplates([]string{"index", "navbar"}, getFuncMapFor("", false))
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

func serverHandler(w http.ResponseWriter, r *http.Request, templateCommonData CommonWebData, servCfg ServerConfig) {
	tmpl, err := parseTemplates([]string{"server", "navbar", "archive-loader"}, getFuncMapFor(servCfg.ServerTag, false))
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	availableLogs := []string{}
	if servCfg.archivesEnabled {
		availableLogs, err = listArchivedLogs(servCfg.ArchiveLogsDirPath, servCfg.ArchiveLogFilenameFormat)
		if err != nil {
			handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
			return
		}
	}

	templateCommonData.ExecDate = time.Now().Format("15:04:05")
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
		ServerLogs:                getServerLogs(servCfg.LogFilePath),
	})
	if err != nil {
		printError(err)
	}
}

func archiveHandler(w http.ResponseWriter, logFile string, templateCommonData CommonWebData, servCfg ServerConfig) {
	tmpl, err := parseTemplates([]string{"archive", "navbar", "archive-loader"}, getFuncMapFor(servCfg.ServerTag, true))
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	availableLogs, err := listArchivedLogs(servCfg.ArchiveLogsDirPath, servCfg.ArchiveLogFilenameFormat)
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	templateCommonData.ExecDate = time.Now().Format("15:04:05")
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
		ServerLogs:                getArchiveLogs(filepath.Join(servCfg.ArchiveLogsDirPath, logFile)),
	})
	if err != nil {
		printError(err)
	}
}

func listArchivesHandler(w http.ResponseWriter, templateCommonData CommonWebData, servCfg ServerConfig) {
	tmpl, err := parseTemplates([]string{"archive", "navbar", "archive-loader"}, getFuncMapFor(servCfg.ServerTag, true))
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	availableLogs, err := listArchivedLogs(servCfg.ArchiveLogsDirPath, servCfg.ArchiveLogFilenameFormat)
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	templateCommonData.ExecDate = time.Now().Format("15:04:05")
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
