package main

import (
	fifo "github.com/foize/go.fifo"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"
)

func TestWatcherWrite(t *testing.T) {
	rand.Seed(time.Now().Unix())

	logFilePath := path.Join(os.TempDir(), "LogRenderer_watcher_test.log")
	logFile, err := os.Create(logFilePath)
	if err != nil {
		t.Fatalf("failed to create log file: %v", err)
	}

	logQueue := fifo.NewQueue()

	go watchServ(logQueue, watchProperties{ // start the watcher
		servName:                  "test-server",
		logFilePath:               logFilePath,
		shouldRewatchOnFileRemove: false,
		delayBeforeRewatch:        0,
	})

	time.Sleep(time.Millisecond) // time for the watcher to set up

	linesCount := rand.Intn(990) + 10 // generate between 10 and 1000 lines of log
	lines := make([]string, linesCount)
	for i := 0; i < linesCount; i++ {
		line := generateLogLine()
		lines[i] = string(line)
		_, err = logFile.Write(line)
		if err != nil {
			t.Fatalf("failed to write logs to file: %v", err)
		}
		time.Sleep(time.Millisecond) // so as not to write too fast
	}

	if linesCount != logQueue.Len() {
		t.Errorf("Invalid number of log lines, expected %d got %d.", linesCount, logQueue.Len())
		t.FailNow()
	}

	for i := 0; logQueue.Len() > 0; i++ {
		event := logQueue.Next().(fileEvent)
		// event type check
		if event.eventType != eventAdd {
			t.Errorf("Invalid event type: expected %q, got %q.", eventAdd, event.eventType)
		}
		// event content check
		line := event.content
		expectedLine := lines[i]
		if line != expectedLine {
			t.Errorf("Log line n°%d should be %q, got %q.", i, expectedLine, line)
		}
	}
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
