package main

import (
	"context"
	"math/rand"
	"strings"
	"testing"
	"time"

	fifo "github.com/foize/go.fifo"
	"github.com/stretchr/testify/assert"
)

func TestUnstackers(t *testing.T) {
	rand.Seed(time.Now().Unix())

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

			outputEvt := <-outputChannel
			if !assert.Equal(t, evt.eventType, outputEvt.Type, "Bad event type") {
				doneChannel <- struct{}{}
				return
			}
			if !assert.Equal(t, "test", outputEvt.Server, "Bad event server") {
				doneChannel <- struct{}{}
				return
			}
			if !assert.Equal(t, evt.content, outputEvt.Content, "Bad event content") {
				doneChannel <- struct{}{}
				return
			}
			if dynamic {
				if !assert.True(t, outputEvt.isDynamic, "Event wasn't marked as dynamic when it should have been.") {
					doneChannel <- struct{}{}
					return
				}
				if !assert.Equal(t, "t", outputEvt.instance, "Bad event instance") {
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
