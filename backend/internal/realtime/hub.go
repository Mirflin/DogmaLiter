package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Event is the envelope pushed to clients. Type identifies what changed; Data
// carries optional context (e.g. the character ID that was updated). Clients
// react by refreshing the relevant slice of state over the existing REST API.
type Event struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

const (
	// keepalive ping interval; also lets us notice dead connections.
	pingInterval = 30 * time.Second
	// per-write deadline.
	writeTimeout = 10 * time.Second
	// buffered outbound messages before a slow client is dropped.
	sendBuffer = 32
)

type client struct {
	gameID string
	conn   *websocket.Conn
	send   chan []byte
}

// Hub keeps the set of live WebSocket connections grouped by game and
// fans out events to every connection of a given game. Safe for concurrent use.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{} // gameID -> set of clients
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.gameID] == nil {
		h.clients[c.gameID] = make(map[*client]struct{})
	}
	h.clients[c.gameID][c] = struct{}{}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.gameID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.gameID)
		}
	}
}

// BroadcastEvent sends an event to every client connected to gameID. It never
// blocks: if a client's buffer is full it is skipped (it will recover via the
// polling fallback). Marshalling failures are silently ignored.
func (h *Hub) BroadcastEvent(gameID, eventType string, data map[string]interface{}) {
	payload, err := json.Marshal(Event{Type: eventType, Data: data})
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[gameID] {
		select {
		case c.send <- payload:
		default:
			// Slow consumer: drop this message rather than block the broadcaster.
		}
	}
}

// Serve registers the connection and runs its write loop until the client
// disconnects or an error occurs. Incoming application messages are discarded
// (all client actions go through REST); CloseRead handles ping/pong and close
// frames and signals disconnect via the returned context.
func (h *Hub) Serve(ctx context.Context, gameID string, conn *websocket.Conn) {
	c := &client{gameID: gameID, conn: conn, send: make(chan []byte, sendBuffer)}
	h.add(c)
	defer h.remove(c)

	readCtx := conn.CloseRead(ctx)

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-readCtx.Done():
			return
		case msg := <-c.send:
			if err := writeWithTimeout(readCtx, conn, websocket.MessageText, msg); err != nil {
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(readCtx, writeTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func writeWithTimeout(ctx context.Context, conn *websocket.Conn, typ websocket.MessageType, msg []byte) error {
	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return conn.Write(writeCtx, typ, msg)
}
