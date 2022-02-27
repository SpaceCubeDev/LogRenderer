package main

import (
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

func (h *Hub) disconnectClient(client *Client, server string) {
	if _, ok := h.clients[client]; ok {
		client.connected = false
		delete(h.clients, client)
		close(client.send)
		if serverTag, serverId, isDynamic := parseWSServer(server); isDynamic { // remove client from dynamic servers
			h.clientsByDynamicServerMutex.Lock()
			defer h.clientsByDynamicServerMutex.Unlock()
			dynamicServersClients := h.clientsByDynamicServer[serverTag][serverId]
			// for instance, instanceClients := range dynamicServersClients {
			for i, c := range dynamicServersClients {
				if c == client {
					h.clientsByDynamicServer[serverTag][serverId] = append(dynamicServersClients[:i], dynamicServersClients[i+1:]...)
					return
				}
			}
			// }
		} else { // remove client from classic servers
			h.clientsByServerMutex.Lock()
			defer h.clientsByServerMutex.Unlock()
			serverClients := h.clientsByServer[server]
			for i, c := range serverClients {
				if c == client {
					h.clientsByServer[server] = append(serverClients[:i], serverClients[i+1:]...)
					return
				}
			}
		}

	}
}

func (h *Hub) run(eventChan <-chan Event) {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = client.server
		case client := <-h.unregister:
			h.disconnectClient(client, h.clients[client])
		case evt := <-eventChan:
			eventMsg := append(evt.Json(), messageSeparator...)
			for _, client := range h.getClientsSubscribedTo(evt) {
				if !client.connected {
					continue
				}
				select {
				case client.send <- eventMsg:
					// Message sent successfully
				default:
					h.disconnectClient(client, evt.Server)
				}
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func (h *Hub) serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		printError(err)
		return
	}

	client := &Client{
		connected: true,
		hub:       h,
		conn:      conn,
		send:      make(chan []byte, 256),
	}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writer()
	go client.readPump()
}

func (h Hub) getClientsSubscribedTo(evt Event) []*Client {
	if evt.isDynamic {
		if instances, found := h.clientsByDynamicServer[evt.Server]; found {
			return instances[evt.instance]
		}
		return []*Client{} // server not found
	}
	return h.clientsByServer[evt.Server]
}
