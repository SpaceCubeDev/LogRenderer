package main

import (
	fifo "github.com/foize/go.fifo"
	"github.com/fsnotify/fsnotify"
	"io"
	"log"
	"os"
	"sync"
)

const bufferSize = 32768

func watchServ(servName, logFilePath string, logQueue *fifo.Queue) {

	shouldRewatch := true
	var filePos int64

	for shouldRewatch {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}

		wg := new(sync.WaitGroup)
		wg.Add(1)

		go func() {
			for event := range watcher.Events {
				if event.Op&fsnotify.Write == fsnotify.Write {
					file, err := os.Open(logFilePath)
					if err != nil {
						log.Fatal(prefix(servName, false), "open:", err)
					}
					stat, err := file.Stat()
					if err != nil {
						log.Fatal(prefix(servName, false), "stat:", err)
					}
					if stat.Size() < filePos {
						logQueue.Add(fileEvent{eventType: eventReset})
						filePos = 0
						continue
					}
					filePos, err = file.Seek(filePos, 0)
					if err != nil {
						log.Fatal(prefix(servName, false), "seek:", err)
					}
					buffer := make([]byte, bufferSize)
					readLength, err := file.Read(buffer)
					filePos += int64(readLength)
					if err != nil {
						if err == io.EOF {
							continue
						}
						log.Fatal(prefix(servName, false), "read:", err)
					}

					if readLength > 0 {
						newData := string(buffer[:readLength])
						logQueue.Add(fileEvent{
							eventType: eventAdd,
							content:   newData,
						})
					}
					if readLength >= bufferSize {
						log.Println("Buffer size is not enough for", readLength)
					}
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Println(prefix(servName, false), "Rename")
					shouldRewatch = true
					wg.Done()
					return
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println(prefix(servName, false), "Remove")
					if err = checkFile(logFilePath); err != nil {
						printError(err)
						shouldRewatch = false
					} else {
						shouldRewatch = true
					}
					wg.Done()
					return
				} else {
					log.Println(prefix(servName, false), "event:", event)
				}
			}
		}()

		err = watcher.Add(logFilePath)
		if err != nil {
			log.Fatal(err)
		}
		wg.Wait()

		err = watcher.Close()
		if err != nil {
			log.Fatal(err)
		}
	}

}
