package main

import (
	"bytes"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	// Whether the client is connected or not
	connected bool

	// The Hub the client is connected to
	hub *Hub

	// The server the client is currently subscribed to
	server string

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *Client) writer() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				printError(err)
				continue
			}
			if !ok {
				// The hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				printError(err)
				continue
			}
			_, _ = w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write(newline)
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				printError(err)
				continue
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		err := c.conn.Close()
		if err != nil {
			printError(err)
		}
	}()
	c.conn.SetReadLimit(maxMessageSize)
	err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		return
	}
	c.conn.SetPongHandler(func(string) error {
		err = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return err
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.handleMessage(string(message))
	}
}

func (c *Client) handleMessage(server string) {
	serverTag, serverId, valid := parseWSServer(server)
	if valid { // Dynamic server (server=>instance)
		if instances, found := c.hub.clientsByDynamicServer[serverTag]; found {
			if clients, found := instances[serverId]; found {
				c.server = server
				c.hub.clientsByDynamicServerMutex.Lock()
				c.hub.clientsByDynamicServer[serverTag][serverId] = append(clients, c)
				c.hub.clientsByDynamicServerMutex.Unlock()
				return
			}
		}
	} else { // Classic server
		if clients, found := c.hub.clientsByServer[server]; found {
			c.server = server
			c.hub.clientsByServerMutex.Lock()
			c.hub.clientsByServer[server] = append(clients, c)
			c.hub.clientsByServerMutex.Unlock()
			return
		}
	}
	debugPrint("Unknown server: " + server)
	c.send <- append(Event{Type: eventError, Message: "Unknown server: " + server}.Json(), messageSeparator...)
}
