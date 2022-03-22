package main

import (
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

//go:embed resources/index.tmpl
var indexHtml string

//go:embed resources/server.tmpl
var serverHtml string

//go:embed resources/archive.tmpl
var archiveHtml string

//go:embed resources/navbar.tmpl
var navbarHtml string

//go:embed resources/archive-loader.tmpl
var archiveLoaderHtml string

//go:embed resources/common-scripts.tmpl
var commonScriptsJs string

//go:embed resources/global.css
var globalCss []byte

//go:embed resources/server.css
var serverCss []byte

//go:embed resources/archive.css
var archiveCss []byte

//go:embed resources/favicon.png
var favicon []byte

func parseTemplates(funcMap template.FuncMap, templateNames ...string) (finalTmpl *template.Template, err error) {
	for _, templateName := range templateNames {
		var templatePtr *string

		switch templateName {
		case "navbar":
			templatePtr = &navbarHtml
		case "index":
			templatePtr = &indexHtml
		case "server":
			templatePtr = &serverHtml
		case "archive-loader":
			templatePtr = &archiveLoaderHtml
		case "archive":
			templatePtr = &archiveHtml
		case "common-scripts":
			templatePtr = &commonScriptsJs
		default:
			err = errors.New("template '" + templateName + "' not found")
			printError(err)
			return finalTmpl.New("error").Parse(errorHtml)
		}

		if finalTmpl == nil {
			finalTmpl, err = template.New(templateName).Funcs(funcMap).Parse(*templatePtr)
			if err != nil {
				printError(err)
				return finalTmpl.New("error").Parse(errorHtml)
			}
			continue
		}

		if templateName == finalTmpl.Name() {
			_, err = finalTmpl.Parse(*templatePtr)
		} else {
			_, err = finalTmpl.New(templateName).Parse(*templatePtr)
		}
		if err != nil {
			return nil, err
		}
	}

	return
}

func serveResource(w http.ResponseWriter, r *http.Request) {
	resourceName := strings.TrimPrefix(r.URL.Path, "/res/")
	var resourcePtr *[]byte

	switch resourceName {
	case "global-css":
		w.Header().Set("Content-Type", "text/css")
		resourcePtr = &globalCss
	case "server-css":
		w.Header().Set("Content-Type", "text/css")
		resourcePtr = &serverCss
	case "archive-css":
		w.Header().Set("Content-Type", "text/css")
		resourcePtr = &archiveCss
	case "favicon-png":
		w.Header().Set("Content-Type", "image/png")
		resourcePtr = &favicon
	default:
		// printError(errors.New("resource '" + resourceName + "' not found"))
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Resource '"+resourceName+"' not found")
		return
	}

	_, err := w.Write(*resourcePtr)
	if err != nil {
		printError(err)
	}
}

func handleTemplateError(w http.ResponseWriter, tmpl *template.Template, statusCode int, err error) {
	printError(err)
	tmplError := tmpl.Execute(w, struct {
		ErrorCode    int
		ErrorStatus  string
		ErrorMessage string
	}{
		ErrorCode:    statusCode,
		ErrorStatus:  http.StatusText(statusCode),
		ErrorMessage: err.Error(),
	})
	if tmplError != nil {
		exitWithError(tmplError)
	}
}

var errorHtml = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Error</title>
</head>
<body style="text-align: center">
	<h1>{{.ErrorCode - .ErrorName}}</h1>
	<hr style="width: 50%"/>
	<h3>{{.ErrorMessage}}</h3>
</body>
</html>
`
