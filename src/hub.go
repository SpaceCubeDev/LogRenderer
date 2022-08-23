package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	newline          = []byte{'\n'}
	space            = []byte{' '}
	messageSeparator = []byte("\n,,,\n")
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients with the server they are subscribed to.
	clients map[*Client]string

	// Registered clients subscribed to every classic server.
	clientsByServer      map[string][]*Client
	clientsByServerMutex *sync.Mutex

	// Registered clients subscribed to every instance of every dynamic server.
	clientsByDynamicServer      map[string]map[string][]*Client
	clientsByDynamicServerMutex *sync.Mutex

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients:                     make(map[*Client]string),
		clientsByServer:             make(map[string][]*Client),
		clientsByServerMutex:        new(sync.Mutex),
		clientsByDynamicServer:      make(map[string]map[string][]*Client),
		clientsByDynamicServerMutex: new(sync.Mutex),
		register:                    make(chan *Client),
		unregister:                  make(chan *Client),
	}
}

func (hub *Hub) disconnectClient(client *Client, server string) {
	if _, ok := hub.clients[client]; ok {
		client.connected = false
		delete(hub.clients, client)
		close(client.send)
		if serverTag, serverId, isDynamic := parseWSServer(server); isDynamic { // remove client from dynamic servers
			hub.clientsByDynamicServerMutex.Lock()
			defer hub.clientsByDynamicServerMutex.Unlock()
			dynamicServersClients := hub.clientsByDynamicServer[serverTag][serverId]
			// for instance, instanceClients := range dynamicServersClients {
			for i, c := range dynamicServersClients {
				if c == client {
					hub.clientsByDynamicServer[serverTag][serverId] = append(dynamicServersClients[:i], dynamicServersClients[i+1:]...)
					return
				}
			}
			// }
		} else { // remove client from classic servers
			hub.clientsByServerMutex.Lock()
			defer hub.clientsByServerMutex.Unlock()
			serverClients := hub.clientsByServer[server]
			for i, c := range serverClients {
				if c == client {
					hub.clientsByServer[server] = append(serverClients[:i], serverClients[i+1:]...)
					return
				}
			}
		}

	}
}

func (hub *Hub) run(eventChan <-chan Event) {
	for {
		select {
		case client := <-hub.register:
			hub.clients[client] = client.server
		case client := <-hub.unregister:
			hub.disconnectClient(client, hub.clients[client])
		case evt := <-eventChan:
			eventMsg := append(evt.Json(), messageSeparator...)
			for _, client := range hub.getClientsSubscribedTo(evt) {
				if !client.connected {
					continue
				}
				select {
				case client.send <- eventMsg:
					// Message sent successfully
				default:
					hub.disconnectClient(client, evt.Server)
				}
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func (hub *Hub) serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		printError(fmt.Errorf("failed to upgrade connection: %v", err))
		return
	}

	client := &Client{
		connected: true,
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, 256),
	}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writer()
	go client.readPump()
}

func (hub *Hub) getClientsSubscribedTo(evt Event) []*Client {
	if evt.isDynamic {
		if instances, found := hub.clientsByDynamicServer[evt.Server]; found {
			return instances[evt.instance]
		}
		return []*Client{} // server not found
	}
	return hub.clientsByServer[evt.Server]
}

func (hub *Hub) sendResetMessage(server, instance string) {
	resetMessage := Event{
		Type:      eventReset,
		Server:    server,
		isDynamic: true,
		instance:  instance,
	}.Json()
	for _, c := range hub.clientsByDynamicServer[server][instance] {
		if c.connected {
			c.send <- resetMessage
		}
	}
}
