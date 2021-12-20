package main

import (
	"fmt"
	"html/template"
	"net/http"
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
}

func createHandlerFor(servCfg ServerConfig, templateCommonData CommonWebData) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		serverHandler(w, r, templateCommonData, servCfg)
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
		http.HandleFunc("/server/"+servCfg.ServerTag, createHandlerFor(servCfg, templateCommonData))
	}

	http.HandleFunc("/server", func(w http.ResponseWriter, r *http.Request) {
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
	tmpl, err := parseTemplates([]string{"index", "navbar"}, template.FuncMap{"getCurrentServer": func() string { return "" }})
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
	tmpl, err := parseTemplates([]string{"server", "navbar"}, template.FuncMap{"getCurrentServer": func() string { return servCfg.ServerTag }})
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	templateCommonData.ExecDate = time.Now().Format("15:04:05")
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
