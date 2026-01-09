package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// Message represents a WebSocket message
type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	// Registered clients mapped by email address
	clients map[string]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to clients for a specific address
	broadcast chan BroadcastMessage

	// Mutex for thread-safe access to clients map
	mu sync.RWMutex

	logger *slog.Logger
}

// BroadcastMessage contains the message and target address
type BroadcastMessage struct {
	Address string
	Message Message
}

// NewHub creates a new WebSocket hub
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan BroadcastMessage, 256),
		logger:     logger,
	}
}

// Run starts the hub and processes register/unregister/broadcast events
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.address] == nil {
				h.clients[client.address] = make(map[*Client]bool)
			}
			h.clients[client.address][client] = true
			h.mu.Unlock()
			h.logger.Info("Client registered", "address", client.address)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.address]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.address)
					}
				}
			}
			h.mu.Unlock()
			h.logger.Info("Client unregistered", "address", client.address)

		case broadcastMsg := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[broadcastMsg.Address]
			h.mu.RUnlock()

			if clients != nil {
				// Convert message to JSON
				messageBytes, err := json.Marshal(broadcastMsg.Message)
				if err != nil {
					h.logger.Error("Failed to marshal broadcast message", "error", err)
					continue
				}

				// Send to all clients subscribed to this address
				for client := range clients {
					select {
					case client.send <- messageBytes:
					default:
						// Client's send buffer is full, close the connection
						h.mu.Lock()
						close(client.send)
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.clients, broadcastMsg.Address)
						}
						h.mu.Unlock()
						h.logger.Warn("Client send buffer full, closing connection", "address", client.address)
					}
				}
			}
		}
	}
}

// BroadcastToAddress sends a message to all clients subscribed to a specific address
func (h *Hub) BroadcastToAddress(address string, message Message) {
	h.broadcast <- BroadcastMessage{
		Address: address,
		Message: message,
	}
}

// GetClientCount returns the number of connected clients for an address
func (h *Hub) GetClientCount(address string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[address])
}
