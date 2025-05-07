package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/mehranredrose/warpture/tunnel-agent/internal/split"
	"github.com/mehranredrose/warpture/tunnel-agent/internal/warp"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
	Status  string      `json:"status,omitempty"`
}

type wsHub struct {
	clients    map[*wsClient]bool
	broadcast  chan WSMessage
	register   chan *wsClient
	unregister chan *wsClient
	mu         sync.RWMutex
}

func newHub() *wsHub {
	return &wsHub{
		clients:    make(map[*wsClient]bool),
		broadcast:  make(chan WSMessage, 64),
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
	}
}

func (h *wsHub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					close(c.send)
					delete(h.clients, c)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *wsHub) Broadcast(msg WSMessage) {
	select {
	case h.broadcast <- msg:
	default:
		log.Warn("[ws] broadcast channel full, dropping message")
	}
}

type wsClient struct {
	conn *websocket.Conn
	send chan WSMessage
	hub  *wsHub
}

func (c *wsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() { ticker.Stop(); c.conn.Close() }()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *wsClient) readPump() {
	defer func() { c.hub.unregister <- c; c.conn.Close() }()
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

type WSHandler struct {
	hub   *wsHub
	warp  *warp.Client
	split *split.Manager
}

func NewWSHandler(w *warp.Client, m *split.Manager) http.Handler {
	hub := newHub()
	go hub.run()
	h := &WSHandler{hub: hub, warp: w, split: m}

	w.OnStatusChange(func(status warp.Status) {
		hub.Broadcast(WSMessage{
			Type:   "status",
			Status: string(status),
			Payload: map[string]interface{}{
				"status": status,
				"stats":  w.GetStats(),
			},
		})
	})

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			hub.mu.RLock()
			hasClients := len(hub.clients) > 0
			hub.mu.RUnlock()
			if hasClients {
				hub.Broadcast(WSMessage{Type: "appUpdate", Payload: m.GetApps()})
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.serveWS)
	return mux
}

func (h *WSHandler) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("[ws] upgrade error: %v", err)
		return
	}
	client := &wsClient{conn: conn, send: make(chan WSMessage, 32), hub: h.hub}
	h.hub.register <- client

	st, _ := h.warp.GetStatus()
	client.send <- WSMessage{
		Type:   "status",
		Status: string(st),
		Payload: map[string]interface{}{
			"status": st,
			"stats":  h.warp.GetStats(),
			"mock":   h.warp.IsMockMode(),
		},
	}
	client.send <- WSMessage{Type: "appUpdate", Payload: h.split.GetApps()}

	go client.writePump()
	client.readPump()
}
