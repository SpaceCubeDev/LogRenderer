package main

import (
	_ "embed"
	"errors"
	"html/template"
	"net/http"
	"strings"
)

//go:embed resources/index.tmpl
var indexHtml string

//go:embed resources/server.tmpl
var serverHtml string

//go:embed resources/global.css
var globalCss []byte

//go:embed resources/server.css
var serverCss []byte

func parseTemplate(templateName string, funcMap template.FuncMap) (*template.Template, error) {
	var templatePtr *string

	switch templateName {
	case "index":
		templatePtr = &indexHtml
	case "server":
		templatePtr = &serverHtml
	default:
		printError(errors.New("template '" + templateName + "' not found"))
		return template.New("error").Parse(errorHtml)
	}

	tmpl, err := template.New(templateName).Funcs(funcMap).Parse(*templatePtr)
	if err != nil {
		printError(err)
		return template.New("error").Parse(errorHtml)
	}
	return tmpl, nil
}

func serveResource(w http.ResponseWriter, r *http.Request) {
	resourceName := strings.TrimPrefix(r.URL.Path, "/res/")
	var resourcePtr *[]byte

	switch resourceName {
	case "global.css":
		w.Header().Set("Content-Type", "text/css")
		resourcePtr = &globalCss
	case "server.css":
		w.Header().Set("Content-Type", "text/css")
		resourcePtr = &serverCss
	default:
		printError(errors.New("resource '" + resourceName + "' not found"))
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
