package main

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWatcherWrite(t *testing.T) {
	rand.Seed(time.Now().Unix())

	logFile, logQueue := newLogFile("test-server", t)

	linesCount := rand.Intn(990) + 10 // generates between 10 and 1000 lines of log
	t.Logf("Testing with %d lines ...", linesCount)
	lines := make([]string, linesCount)
	for i := 0; i < linesCount; i++ {
		line := generateLogLine()
		lines[i] = string(line)
		_, err := logFile.Write(line)
		if err != nil {
			t.Fatalf("failed to write logs to file: %v", err)
		}
		time.Sleep(3 * time.Millisecond) // so as not to write too fast
	}

	/*if linesCount != logQueue.Len() {
		t.Fatalf("Invalid number of log lines, expected %d got %d.", linesCount, logQueue.Len())
	}*/
	assert.Equal(t, linesCount, logQueue.Len(), "Invalid number of log lines")

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
