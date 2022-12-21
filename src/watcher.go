package main

import (
	"io"
	"log"
	"os"
	"sync"
	"time"

	fifo "github.com/foize/go.fifo"
	"github.com/fsnotify/fsnotify"
)

const bufferSize = 32768

func watchServ(logQueue *fifo.Queue, properties watchProperties) {

	shouldRewatch := true

	stat, err := os.Stat(properties.logFilePath)
	if err != nil {
		log.Fatal(prefix(properties.servName, true), "stat: ", err)
	}
	filePos := stat.Size()

	for shouldRewatch {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(prefix(properties.servName, true), err)
		}

		wg := new(sync.WaitGroup)
		wg.Add(1)

		go func() {
			for event := range watcher.Events {
				if event.Op&fsnotify.Write == fsnotify.Write {
					file, err := os.Open(properties.logFilePath)
					if err != nil {
						log.Fatal(prefix(properties.servName, true), "open: ", err)
					}
					stat, err := file.Stat()
					if err != nil {
						log.Fatal(prefix(properties.servName, true), "stat: ", err)
					}
					if stat.Size() < filePos {
						logQueue.Add(fileEvent{eventType: eventReset})
						filePos = 0
						_ = file.Close()
						continue
					}
					filePos, err = file.Seek(filePos, 0)
					if err != nil {
						log.Fatal(prefix(properties.servName, true), "seek: ", err)
					}
					buffer := make([]byte, bufferSize)
					readLength, err := file.Read(buffer)
					filePos += int64(readLength)
					if err != nil {
						if err == io.EOF {
							_ = file.Close()
							continue
						}
						log.Fatal(prefix(properties.servName, true), "read: ", err)
					}

					if readLength > 0 {
						newData := string(buffer[:readLength])
						logQueue.Add(fileEvent{
							eventType: eventAdd,
							content:   newData,
						})
					}
					if readLength >= bufferSize {
						log.Println(prefix(properties.servName, false), "Buffer size is not enough, missing", readLength-bufferSize, "of length")
					}
					_ = file.Close()
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Println(prefix(properties.servName, false), "Rename")
					shouldRewatch = properties.shouldRewatchOnFileRemove
					wg.Done()
					return
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println(prefix(properties.servName, false), "Remove")
					if properties.shouldRewatchOnFileRemove {
						if err = checkFile(properties.logFilePath); err != nil {
							printError(err)
							shouldRewatch = false
						} else {
							shouldRewatch = true
						}
					} else {
						shouldRewatch = false
					}
					wg.Done()
					return
				} else {
					log.Println(prefix(properties.servName, false), "event:", event)
				}
			}
		}()

		err = watcher.Add(properties.logFilePath)
		if err != nil {
			log.Fatal(prefix(properties.servName, true), "add watcher: ", err)
		}

		wg.Wait()

		err = watcher.Close()
		if err != nil {
			log.Fatal(prefix(properties.servName, true), "close watcher: ", err)
		}

		if shouldRewatch {
			time.Sleep(properties.delayBeforeRewatch)
		}
	}

}

type watchProperties struct {
	servName                  string
	logFilePath               string
	shouldRewatchOnFileRemove bool
	delayBeforeRewatch        time.Duration
}
