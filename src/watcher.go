package main

import (
	"fmt"
	fifo "github.com/foize/go.fifo"
	"github.com/fsnotify/fsnotify"
	"io"
	"log"
	"os"
	"sync"
)

const bufferSize = 16384

func watchServ(logFilePath string, logQueue *fifo.Queue) {

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
						log.Fatal("open:", err)
					}
					stat, err := file.Stat()
					if err != nil {
						log.Fatal("stat:", err)
					}
					if stat.Size() < filePos {
						logQueue.Add(fileEvent{eventType: eventReset})
						filePos = 0
						continue
					}
					filePos, err = file.Seek(filePos, 0)
					if err != nil {
						log.Fatal("seek:", err)
					}
					buffer := make([]byte, bufferSize)
					readLength, err := file.Read(buffer)
					filePos += int64(readLength)
					if err != nil {
						if err == io.EOF {
							continue
						}
						log.Fatal("read:", err)
					}

					if readLength > 0 {
						newData := string(buffer[:readLength])
						logQueue.Add(fileEvent{
							eventType: eventAdd,
							content:   newData,
						})
					}
					if readLength >= bufferSize {
						log.Println("Buffer size is not enough !")
					}
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					fmt.Println("Rename")
					shouldRewatch = true
					wg.Done()
					return
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					fmt.Println("Remove")
					if err = checkFile(logFilePath); err != nil {
						printError(err)
						shouldRewatch = false
					} else {
						shouldRewatch = true
					}
					wg.Done()
					return
				} else {
					log.Println("event:", event)
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
