package http

import (
	"encoding/json"
	"net/http"
	"sync"

	"mock-my-mta/log"

	"github.com/gorilla/websocket"
)

// WSEvent is a real-time event pushed to connected WebSocket clients.
type WSEvent struct {
	Type    string      `json:"type"`    // "new_email", "delete_email", "delete_all"
	Payload interface{} `json:"payload"` // event-specific data
}

// wsHub manages connected WebSocket clients and broadcasts events.
type wsHub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

var hub = &wsHub{
	clients: make(map[*websocket.Conn]bool),
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // allow all origins for dev tool
}

// handleWebSocket upgrades the HTTP connection and registers the client.
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Logf(log.ERROR, "websocket upgrade failed: %v", err)
		return
	}

	hub.mu.Lock()
	hub.clients[conn] = true
	hub.mu.Unlock()

	log.Logf(log.DEBUG, "websocket client connected (%d total)", len(hub.clients))

	// Keep connection alive — read loop (discards incoming messages)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	hub.mu.Lock()
	delete(hub.clients, conn)
	hub.mu.Unlock()
	conn.Close()
	log.Logf(log.DEBUG, "websocket client disconnected (%d remaining)", len(hub.clients))
}

// BroadcastEvent sends an event to all connected WebSocket clients.
func BroadcastEvent(eventType string, payload interface{}) {
	event := WSEvent{Type: eventType, Payload: payload}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for conn := range hub.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Logf(log.DEBUG, "websocket write error: %v", err)
			conn.Close()
			delete(hub.clients, conn)
		}
	}
}
