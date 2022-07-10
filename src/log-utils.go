package main

import (
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

type archiveEntry struct {
	Name     string
	FullName string
	modTime  time.Time
	Date     string
}

func listArchivedLogFiles(logsDirRootPath, logsFilePattern string) ([]archiveEntry, error) {
	matches, err := filepath.Glob(filepath.Join(logsDirRootPath, logsFilePattern))
	if err != nil {
		return []archiveEntry{}, err
	}

	var entries []archiveEntry
	for _, entry := range matches {
		info, err := os.Stat(entry)
		if err != nil {
			return []archiveEntry{}, err
		}
		if info.IsDir() {
			continue
		}
		modTime := info.ModTime()
		newEntry := archiveEntry{
			Name:     filepath.Base(entry),
			FullName: filePathEscape(strings.TrimPrefix(entry, logsDirRootPath)),
			modTime:  modTime,
			Date:     modTime.Format("02-01 15:04:05"),
		}
		inserted := false
		for i, e := range entries {
			if e.modTime.Before(modTime) {
				entries = append(entries[:i], append([]archiveEntry{newEntry}, entries[i:]...)...)
				inserted = true
				break
			}
		}
		if !inserted {
			entries = append(entries, newEntry)
		}
	}

	return entries, nil
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
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

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
