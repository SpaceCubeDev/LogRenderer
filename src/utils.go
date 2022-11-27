package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"
)

const defaultMaxLinesCount = 500

func printError(err error) {
	fmt.Println("\n/!\\    Error    /!\\")
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("- - - - - - - - - -")
	fmt.Println(err)
}

func exitWithError(err error) {
	fmt.Println("\n/!\\ Fatal Error /!\\")
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("- - - - - - - - - -")
	fmt.Println(err)
	os.Exit(1)
}

func debugPrint(msg string) {
	if doDebug {
		log.Println("[DEBUG]", msg)
	}
}

func checkFile(filePath string) error {
	fileStat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("log file at '" + filePath + "' not found")
		}
		return err
	}
	if fileStat.IsDir() {
		return errors.New("log file at '" + filePath + "' is a directory")
	}
	return nil
}

func checkDir(dirPath string) error {
	fileStat, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("directory at '" + dirPath + "' not found")
		}
		return err
	}
	if !fileStat.IsDir() {
		return errors.New("file at '" + dirPath + "' must be a directory")
	}
	return nil
}

func prefix(servName string, endingSpace bool) string {
	space := ""
	if endingSpace {
		space = " "
	}
	return fmt.Sprintf("[%s]%s", servName, space)
}

func prettier(w http.ResponseWriter, message string, data any, status int) {
	if data == nil {
		data = struct{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
		Data    any    `json:"data"`
	}{
		Message: message,
		Data:    data,
	})
	if err != nil {
		printError(fmt.Errorf("failed to marshal http response to json: %v", err))
	}
}

// extractMaxLinesCount returns the maximum number of log lines wanted by the client
func extractMaxLinesCount(r *http.Request) int {
	cookie, err := r.Cookie("max-lines-count")
	if err != nil { // cookie not found
		return 0
	}
	maxLines, _ := strconv.Atoi(cookie.Value)
	if maxLines == 0 {
		cookie.Value = strconv.Itoa(defaultMaxLinesCount)
		return defaultMaxLinesCount
	}
	return maxLines
}

func findAllGroups(re *regexp.Regexp, str string) map[string]string {
	results := make(map[string]string)
	matches := re.FindStringSubmatch(str)
	if matches == nil {
		return map[string]string{}
	}
	for i, groupName := range re.SubexpNames() {
		if i != 0 && groupName != "" {
			results[groupName] = matches[i]
		}
	}
	return results
}

// filePathEscape encodes the given file path into base64, and then makes it url-friendly
func filePathEscape(filePath string) string {
	return url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(filePath)))
}

func filePathUnescape(filePath string) string {
	unescaped, err := url.QueryUnescape(filePath)
	if err != nil {
		log.Println("Failed to unescape string:", err)
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(unescaped)
	if err != nil {
		log.Println("Failed to decode base64:", err)
		return ""
	}
	return string(decoded)
}
