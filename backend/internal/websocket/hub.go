package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/instantmeet/instantmeet/backend/internal/auth"
)

type Event struct {
	Type      string `json:"type"`
	MeetingID string `json:"meetingId,omitempty"`
	UserID    string `json:"userId,omitempty"`
	Payload   any    `json:"payload,omitempty"`
}

type Client struct {
	conn              *websocket.Conn
	userID, meetingID string
	send              chan []byte
}
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
	auth  *auth.Service
}

func NewHub(a *auth.Service) *Hub { return &Hub{rooms: map[string]map[*Client]struct{}{}, auth: a} }

var upgrader = websocket.Upgrader{
	CheckOrigin:    func(r *http.Request) bool { return true },
	ReadBufferSize: 1024, WriteBufferSize: 1024,
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, err := h.auth.UserFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", 401)
		return
	}
	meetingID := r.URL.Query().Get("meetingId")
	if meetingID == "" {
		http.Error(w, "meetingId required", 400)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &Client{conn: conn, userID: user.ID, meetingID: meetingID, send: make(chan []byte, 32)}
	h.register(c)
	go h.writePump(c)
	h.readPump(c)
}

func (h *Hub) Broadcast(meetingID string, event Event) {
	data, _ := json.Marshal(event)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[meetingID] {
		select {
		case c.send <- data:
		default:
		}
	}
}

func (h *Hub) readPump(c *Client) {
	defer h.unregister(c)
	c.conn.SetReadLimit(32 << 10)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error { return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)) })
	for {
		var event Event
		if err := c.conn.ReadJSON(&event); err != nil {
			break
		}
		event.MeetingID, event.UserID = c.meetingID, c.userID
		h.Broadcast(c.meetingID, event)
	}
}

func (h *Hub) writePump(c *Client) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[c.meetingID] == nil {
		h.rooms[c.meetingID] = map[*Client]struct{}{}
	}
	h.rooms[c.meetingID][c] = struct{}{}
}
func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	if room := h.rooms[c.meetingID]; room != nil {
		delete(room, c)
		if len(room) == 0 {
			delete(h.rooms, c.meetingID)
		}
	}
	close(c.send)
	h.mu.Unlock()
	_ = c.conn.Close()
	slog.Debug("websocket disconnected", "meeting", c.meetingID, "user", c.userID)
}
