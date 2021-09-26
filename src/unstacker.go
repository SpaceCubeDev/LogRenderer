package main

import (
	fifo "github.com/foize/go.fifo"
	"math"
	"strings"
	"time"
)

const sendInterval = 5 * time.Millisecond

func unstack(server string, logQueue *fifo.Queue, output chan Event) {
	for {
		startTime := time.Now()
		if logQueue.Len() > 0 {
			event := logQueue.Next().(fileEvent)
			switch event.eventType {
			case eventAdd:
				newLogs := strings.Trim(event.content, "\n")
				if len(newLogs) > 0 {
					for _, log := range strings.Split(newLogs, "\n") {
						startTime = time.Now()
						output <- Event{
							Type:    eventAdd,
							Server:  server,
							Content: log,
						}
						// fmt.Println("Event:", event)
						sleepDuration := sendInterval - time.Since(startTime)
						time.Sleep(time.Duration(math.Abs(float64(sleepDuration))))
					}
				}
				continue
			case eventReset:
				output <- Event{
					Type:   eventReset,
					Server: server,
				}
			}
		}
		time.Sleep(sendInterval - time.Since(startTime)) // yeeessssss
	}
}
