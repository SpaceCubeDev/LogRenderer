package main

import (
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	fifo "github.com/foize/go.fifo"
)

const testAddr = ":8181"

func newLogFile(serverTag string, t *testing.T) (*os.File, *fifo.Queue) {
	logFilePath := path.Join(os.TempDir(), "LogRenderer_watcher_test.log")
	logFile, err := os.Create(logFilePath)
	if err != nil {
		t.Fatalf("failed to create log file: %v", err)
	}

	logQueue := fifo.NewQueue()

	go watchServ(logQueue, watchProperties{ // start the watcher
		servName:                  serverTag,
		logFilePath:               logFilePath,
		shouldRewatchOnFileRemove: false,
		delayBeforeRewatch:        0,
	})
	time.Sleep(time.Millisecond) // time for the watcher to set up
	return logFile, logQueue
}

func generateLogLine() []byte {
	lineLength := rand.Intn(990) + 10 // generate a line of log with a length between 10 and 1000
	line := make([]byte, lineLength)
	for i := 0; i < lineLength-1; i++ {
		c := byte(rand.Int31n(95) + 32) // generate a rune from the character with index 32 (space) to index 126 (~)
		line[i] = c
	}
	line[lineLength-1] = byte('\n') // add a carrage return at the end of the line
	return line
}
