package websocket

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	// Registered clients mapped by deployment ID
	clients    map[string]map[*websocket.Conn]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mutex      sync.Mutex
}

type Client struct {
	Hub          *Hub
	Conn         *websocket.Conn
	DeploymentID string
}

type Message struct {
	DeploymentID string
	Content      []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*websocket.Conn]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			if _, ok := h.clients[client.DeploymentID]; !ok {
				h.clients[client.DeploymentID] = make(map[*websocket.Conn]bool)
			}
			h.clients[client.DeploymentID][client.Conn] = true
			h.mutex.Unlock()
			log.Printf("Client connected to deployment logs: %s", client.DeploymentID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client.DeploymentID]; ok {
				if _, ok := h.clients[client.DeploymentID][client.Conn]; ok {
					delete(h.clients[client.DeploymentID], client.Conn)
					client.Conn.Close()
					if len(h.clients[client.DeploymentID]) == 0 {
						delete(h.clients, client.DeploymentID)
					}
				}
			}
			h.mutex.Unlock()
			log.Printf("Client disconnected from deployment logs: %s", client.DeploymentID)

		case message := <-h.broadcast:
			h.mutex.Lock()
			if clients, ok := h.clients[message.DeploymentID]; ok {
				for conn := range clients {
					err := conn.WriteMessage(websocket.TextMessage, message.Content)
					if err != nil {
						log.Printf("Error writing to websocket: %v", err)
						conn.Close()
						delete(clients, conn)
					}
				}
			}
			h.mutex.Unlock()
		}
	}
}

func (h *Hub) RegisterClient(conn *websocket.Conn, deploymentID string) {
	client := &Client{Hub: h, Conn: conn, DeploymentID: deploymentID}
	h.register <- client
}

func (h *Hub) BroadcastLog(deploymentID string, logMsg string) {
	h.broadcast <- &Message{
		DeploymentID: deploymentID,
		Content:      []byte(logMsg),
	}
}
