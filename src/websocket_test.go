package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	logging "github.com/sacOO7/go-logger"
	"github.com/sacOO7/gowebsocket"
	"github.com/stretchr/testify/assert"
)

func TestWebSocket(t *testing.T) {
	rand.Seed(time.Now().Unix())

	serverTag := "test-server"

	outputChannel := make(chan Event, 16)

	hub := newHub()
	hub.clientsByServer[serverTag] = []*Client{}
	go hub.run(outputChannel)

	logFile, logQueue := newLogFile(serverTag, t)
	go unstack(serverTag, logQueue, outputChannel)

	muxServer := http.NewServeMux()
	muxServer.HandleFunc("/ws", hub.serveWs)
	httpServer := http.Server{
		Addr:    testAddr,
		Handler: muxServer,
	}
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			t.Error("Failed to start http server:", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	connected := make(chan gowebsocket.Socket, 1)
	expectedLogLinesChan := make(chan string)
	wsClient := gowebsocket.New("ws://localhost" + testAddr + "/ws")
	wsClient.GetLogger().SetLevel(logging.OFF)

	testHasEnded := false
	stopOne := new(sync.Once)
	stopFn := func() {
		stopOne.Do(func() {
			testHasEnded = true
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			wsClient.Close()
			err := httpServer.Shutdown(ctx)
			if err != nil {
				t.Log("Failed to shutdown http server:", err)
			}
			cancel()
		})
	}
	defer stopFn()
	exitChan := make(chan struct{}, 1)

	wsClient.OnConnected = func(ws gowebsocket.Socket) {
		connected <- ws
	}

	wsClient.OnConnectError = func(err error, ws gowebsocket.Socket) {
		log.Println("Received connect error ", err)
	}

	wsClient.OnTextMessage = func(message string, ws gowebsocket.Socket) {
		message = strings.TrimSuffix(message, string(messageSeparator))
		decodedMessage := make([]byte, base64.StdEncoding.DecodedLen(len(message)))
		n, err := base64.StdEncoding.Decode(decodedMessage, []byte(message))
		if err != nil {
			t.Error("Failed to decode received event:", err)
			exitChan <- struct{}{}
			<-expectedLogLinesChan // releasing the value to unblock the channel
			return
		}

		var receivedEvt Event
		err = json.Unmarshal(decodedMessage[:n], &receivedEvt)
		if err != nil {
			t.Error("Failed to unmarshal received event:", err)
			exitChan <- struct{}{}
			<-expectedLogLinesChan // releasing the value to unblock the channel
			return
		}

		assert.Equal(t, eventAdd, receivedEvt.Type, "Incorrect event type.")
		assert.Equal(t, serverTag, receivedEvt.Server, "Incorrect event server.")
		assert.Equal(t, <-expectedLogLinesChan, receivedEvt.Content, "Incorrect event content.")
		assert.Equal(t, "", receivedEvt.Message, "Incorrect event message.")
	}

	wsClient.OnBinaryMessage = func(data []byte, ws gowebsocket.Socket) {
		log.Println("Received binary data ", data)
	}

	dco := new(sync.Once)
	wsClient.OnDisconnected = func(err error, ws gowebsocket.Socket) {
		if testHasEnded {
			return
		}
		dco.Do(func() {
			t.Error("WebSocket connection has been closed by the server.")
			exitChan <- struct{}{}
		})
	}

	wsClient.Connect()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var ws gowebsocket.Socket
	select {
	case ws = <-connected:
		t.Log("WS connected")
		break
	case <-ctx.Done():
		t.Fatal("WS connection timed out.")
	}

	ws.SendText(serverTag)
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 10; i++ {
		select {
		case <-exitChan:
			t.FailNow()
		default:
			logLine := generateLogLine()
			_, err := logFile.Write(logLine)
			if err != nil {
				t.Fatal("Failed to write to log file:", err)
			}
			expectedLogLinesChan <- strings.TrimSuffix(string(logLine), "\n")
		}
	}
}
