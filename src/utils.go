package main

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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

func prettier(w http.ResponseWriter, message string, data interface{}, status int) {
	if data == nil {
		data = struct{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(struct {
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}{
		Message: message,
		Data:    data,
	})
	if err != nil {
		printError(fmt.Errorf("failed to marshal http response to json: %v", err))
	}
}

func extractMaxLinesCount(r *http.Request) int {
	cookie, err := r.Cookie("max-lines-count")
	if err != nil { // cookie not found
		return 0
	}
	maxLines, _ := strconv.Atoi(cookie.Value)
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

func getServerLogs(filePath string, limit int) []string {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		printError(err)
		return []string{"Error while reading log file: " + err.Error()}
	}

	lines := strings.Split(string(fileContent), "\n")
	if limit > 0 && len(lines) > limit {
		return lines[len(lines)-limit-1:]
	}
	return lines
}

func listArchivedLogFiles(logsDirPath, logFilePattern string) ([]string, error) {
	entries, err := os.ReadDir(logsDirPath)
	if err != nil {
		return []string{}, err // TODO is not exist special case
	}

	var validEntries []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		match, err := filepath.Match(logFilePattern, entry.Name())
		if err != nil {
			return []string{}, err
		}
		if match {
			validEntries = append(validEntries, entry.Name())
		}
	}

	for i, j := 0, len(validEntries)-1; i < j; i, j = i+1, j-1 {
		validEntries[i], validEntries[j] = validEntries[j], validEntries[i]
	}

	return validEntries, nil
}

func listDynamicArchivedLogFiles(logFilePattern, serverId string) ([]string, error) {
	logFilePath := strings.ReplaceAll(logFilePattern, "%id%", serverId)
	logFilesPaths, err := filepath.Glob(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("invalid archive log file pattern: %v", err)
	}
	return logFilesPaths, nil
}

func getArchiveLogs(logsFilePath string, limit int) []string {
	uncompressed, err := uncompress(logsFilePath)
	if err != nil {
		err = errors.New("Error while uncompressing archived log file: " + err.Error())
		printError(err)
		return []string{err.Error()}
	}

	lines := strings.Split(string(uncompressed), "\n")
	if limit > 0 && len(lines) > limit {
		return lines[len(lines)-limit-1:]
	}
	return lines
}

func uncompress(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return nil, err
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, errors.New("failed to seek archived log file: " + err.Error())
	}

	contentType := http.DetectContentType(buf[:n])
	if i := strings.Index(contentType, ";"); i >= 0 {
		contentType = contentType[:i]
	}

	switch contentType {
	case "text/plain":
		return io.ReadAll(file)
	case "application/x-gzip":
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		return io.ReadAll(gzReader)
	default:
		return io.ReadAll(file)
	}
}
