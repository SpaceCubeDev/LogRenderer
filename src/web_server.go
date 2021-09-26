package main

import (
	"fmt"
	"html/template"
	"net/http"
)

func createHandlerFor(servCfg ServerConfig, serverNames map[string]string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		serverHandler(w, r, servCfg, serverNames)
	}
}

func startServer(address string, servers map[string]ServerConfig, outputChannel chan Event) error {
	hub := newHub()
	go hub.run(outputChannel)

	serverNames := map[string]string{}
	// register a path for every server
	for server, servCfg := range servers {
		serverNames[server] = servCfg.DisplayName
		hub.clientsByServer[server] = []*Client{}
		http.HandleFunc("/server/"+server, createHandlerFor(servCfg, serverNames))
		/*http.HandleFunc("/server/"+server, func(w http.ResponseWriter, r *http.Request) {
			serverHandler(w, r, servCfg.clone(), hub, serverNames)
		})*/
	}

	http.HandleFunc("/server", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
		} else {
			indexHandler(w, r, serverNames)
		}
	})

	http.HandleFunc("/ws", hub.serveWs)

	http.HandleFunc("/res/", serveResource)

	fmt.Println("Starting web server on", address, "...")

	return http.ListenAndServe(address, nil)
	/*time.Sleep(5 * time.Second)
	return nil*/
}

func indexHandler(w http.ResponseWriter, r *http.Request, servers map[string]string) {
	tmpl, err := parseTemplate("index", template.FuncMap{})
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}
	err = tmpl.Execute(w, struct {
		Servers map[string]string
	}{
		Servers: servers,
	})
	if err != nil {
		printError(err)
	}
}

func serverHandler(w http.ResponseWriter, r *http.Request, servCfg ServerConfig, servers map[string]string) {
	tmpl, err := parseTemplate("server", template.FuncMap{"getCurrentServer": func() string { return servCfg.server }})
	if err != nil {
		handleTemplateError(w, tmpl, http.StatusInternalServerError, err)
		return
	}

	err = tmpl.Execute(w, struct {
		Servers                   map[string]string
		Server                    string
		ServerDisplayName         string
		SyntaxHighlightingRegexps SyntaxHighlightingConfig
		ServerLogs                []string
	}{
		Servers:                   servers,
		Server:                    servCfg.server,
		ServerDisplayName:         servCfg.DisplayName,
		SyntaxHighlightingRegexps: servCfg.SyntaxHighlightingRegexps,
		ServerLogs:                getServerLogs(servCfg.LogFilePath),
	})
	if err != nil {
		printError(err)
	}
}
