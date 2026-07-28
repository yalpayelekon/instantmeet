package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/instantmeet/instantmeet/backend/internal/auth"
	"github.com/instantmeet/instantmeet/backend/internal/meeting"
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
	mu          sync.RWMutex
	rooms       map[string]map[*Client]struct{}
	auth        *auth.Service
	meetings    *meeting.Store
	frontendURL string
}

func NewHub(a *auth.Service, store *meeting.Store, frontendURL string) *Hub {
	return &Hub{
		rooms:       map[string]map[*Client]struct{}{},
		auth:        a,
		meetings:    store,
		frontendURL: frontendURL,
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, err := h.auth.UserFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	meetingID := r.URL.Query().Get("meetingId")
	if meetingID == "" {
		http.Error(w, "meetingId required", http.StatusBadRequest)
		return
	}
	m, ok := h.meetings.Get(meetingID)
	if !ok {
		http.Error(w, "meeting not found", http.StatusNotFound)
		return
	}
	if m.Participants[user.ID] == nil && m.WaitingRoom[user.ID] == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin:    h.checkOrigin,
		ReadBufferSize: 1024, WriteBufferSize: 1024,
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

func (h *Hub) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	allowed, err := url.Parse(h.frontendURL)
	if err != nil {
		return false
	}
	got, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return got.Scheme == allowed.Scheme && got.Host == allowed.Host
}

func (h *Hub) Broadcast(meetingID string, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
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
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
		// Server→client only: discard any client payload; do not rebroadcast.
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
		if _, ok := room[c]; ok {
			delete(room, c)
			close(c.send)
		}
		if len(room) == 0 {
			delete(h.rooms, c.meetingID)
		}
	}
	h.mu.Unlock()
	_ = c.conn.Close()
	slog.Debug("websocket disconnected", "meeting", c.meetingID, "user", c.userID)
}
