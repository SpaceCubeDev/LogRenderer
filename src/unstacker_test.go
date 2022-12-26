package main

import (
	"context"
	"math/rand"
	"strings"
	"testing"
	"time"

	fifo "github.com/foize/go.fifo"
)

func TestUnstackers(t *testing.T) {
	const iters = 1000
	logQueue := fifo.NewQueue()
	dynamicLogQueue := fifo.NewQueue()
	outputChannel := make(chan Event, 16)
	stop := false

	go unstack("test", logQueue, outputChannel)
	go unstackDynamic("test", "t", dynamicLogQueue, outputChannel, &stop)

	doneChannel := make(chan struct{})

	const durationThreshold = iters * time.Millisecond * 10
	ctx, cancel := context.WithTimeout(context.Background(), durationThreshold)
	defer cancel()

	go func() {
		for i := 0; i < iters; i++ {
			var evt fileEvent

			if rand.Intn(iters/10) == 1 {
				evt = fileEvent{
					eventType: eventReset,
				}
			} else {
				evt = fileEvent{
					eventType: eventAdd,
					content:   strings.Trim(string(generateLogLine()), "\n"),
				}
			}

			dynamic := rand.Intn(2) == 1
			if dynamic {
				dynamicLogQueue.Add(evt)
			} else {
				logQueue.Add(evt)
			}

			chanEvt := <-outputChannel
			if chanEvt.Type != evt.eventType {
				t.Errorf("Bad event type: want %s, got %s.", evt.eventType, chanEvt.Type)
				doneChannel <- struct{}{}
				return
			}
			if chanEvt.Server != "test" {
				t.Errorf("Bad event server: want %q, got %q.", "test", chanEvt.Server)
				doneChannel <- struct{}{}
				return
			}
			if chanEvt.Content != evt.content {
				t.Errorf("Bad event content: want %q, got %q.", evt.content, chanEvt.Content)
				doneChannel <- struct{}{}
				return
			}
			if dynamic {
				if !chanEvt.isDynamic {
					t.Errorf("Event wasn't marked as dynamic when it should have been.")
					doneChannel <- struct{}{}
					return
				}
				if chanEvt.instance != "t" {
					t.Errorf("Bad event instance: want %q, got %q.", "t", chanEvt.instance)
					doneChannel <- struct{}{}
					return
				}
			}
		}
		doneChannel <- struct{}{}
	}()

	select {
	case <-doneChannel:
		break
	case <-ctx.Done():
		t.Fatalf("Test duration has reached the threshold (%s)", durationThreshold)
	}
}
