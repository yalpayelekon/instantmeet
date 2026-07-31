package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/instantmeet/instantmeet/backend/internal/auth"
	"github.com/instantmeet/instantmeet/backend/internal/meeting"
	"github.com/instantmeet/instantmeet/backend/internal/models"
	ws "github.com/instantmeet/instantmeet/backend/internal/websocket"
)

// MediaTokens mints LiveKit join grants after admission and cleans up SFU rooms.
type MediaTokens interface {
	Token(room string, user models.User, canPublish bool) (string, error)
	PublicURL() string
	DeleteRoom(ctx context.Context, room string) error
}

type API struct {
	meetings *meeting.Store
	hub      *ws.Hub
	livekit  MediaTokens
}

func New(store *meeting.Store, hub *ws.Hub, livekit MediaTokens) *API {
	return &API{store, hub, livekit}
}

func (a *API) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/me", a.me)
	r.Post("/meetings", a.create)
	r.Route("/meetings/{id}", func(r chi.Router) {
		r.Get("/", a.get)
		r.Post("/join", a.join)
		r.Post("/leave", a.leave)
		r.Post("/admit", a.admit)
		r.Post("/reject", a.reject)
		r.Post("/remove", a.remove)
		r.Post("/mute", a.mute)
		r.Post("/chat", a.chat)
		r.Post("/media", a.media)
		r.Post("/end", a.end)
	})
	return r
}

func (a *API) me(w http.ResponseWriter, r *http.Request) { write(w, 200, auth.User(r)) }
func (a *API) create(w http.ResponseWriter, r *http.Request) {
	m := a.meetings.Create(auth.User(r))
	write(w, 201, map[string]any{"meeting": m, "url": "/meet/" + m.ID})
}
func (a *API) get(w http.ResponseWriter, r *http.Request) {
	m, ok := a.meetings.Get(chi.URLParam(r, "id"))
	if !ok {
		problem(w, 404, "meeting not found")
		return
	}
	write(w, 200, publicMeeting(m, auth.User(r).ID))
}
func (a *API) join(w http.ResponseWriter, r *http.Request) {
	user := auth.User(r)
	id := chi.URLParam(r, "id")
	m, err := a.meetings.Update(id, func(m *models.Meeting) error {
		if len(m.Participants)+len(m.WaitingRoom) >= 100 {
			return errors.New("meeting is full")
		}
		if user.ID == m.HostID {
			m.Participants[user.ID] = participant(user, true)
			m.State = models.MeetingActive
		} else if m.Participants[user.ID] == nil {
			m.WaitingRoom[user.ID] = &models.WaitingParticipant{Participant: *participant(user, false), RequestedAt: time.Now().UTC()}
		}
		return nil
	})
	if err != nil {
		problem(w, status(err), err.Error())
		return
	}
	a.hub.Broadcast(id, ws.Event{Type: "meeting.updated", Payload: broadcastMeeting(m)})
	response := map[string]any{"status": "waiting", "meeting": publicMeeting(m, user.ID)}
	if m.Participants[user.ID] != nil {
		token, err := a.livekit.Token(m.LiveKitRoom, user, true)
		if err != nil {
			problem(w, http.StatusBadGateway, "failed to issue media token")
			return
		}
		response["status"], response["livekitToken"], response["livekitUrl"] = "admitted", token, a.livekit.PublicURL()
	}
	write(w, 200, response)
}
func (a *API) admit(w http.ResponseWriter, r *http.Request)  { a.waitingAction(w, r, true) }
func (a *API) reject(w http.ResponseWriter, r *http.Request) { a.waitingAction(w, r, false) }
func (a *API) waitingAction(w http.ResponseWriter, r *http.Request, admit bool) {
	host := auth.User(r)
	id := chi.URLParam(r, "id")
	var body struct {
		UserID string `json:"userId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	m, err := a.meetings.Update(id, func(m *models.Meeting) error {
		if m.HostID != host.ID {
			return errors.New("host only")
		}
		waiting := m.WaitingRoom[body.UserID]
		if waiting == nil {
			return errors.New("participant not waiting")
		}
		delete(m.WaitingRoom, body.UserID)
		if admit {
			p := waiting.Participant
			p.JoinedAt = time.Now().UTC()
			m.Participants[body.UserID] = &p
			m.State = models.MeetingActive
		}
		return nil
	})
	if err != nil {
		problem(w, status(err), err.Error())
		return
	}
	event := "participant.rejected"
	if admit {
		event = "participant.admitted"
	}
	a.hub.Broadcast(id, ws.Event{Type: event, UserID: body.UserID, Payload: broadcastMeeting(m)})
	write(w, 200, publicMeeting(m, host.ID))
}
func (a *API) leave(w http.ResponseWriter, r *http.Request) {
	user := auth.User(r)
	id := chi.URLParam(r, "id")
	destroy := false
	m, err := a.meetings.Update(id, func(m *models.Meeting) error {
		delete(m.Participants, user.ID)
		delete(m.WaitingRoom, user.ID)
		destroy = len(m.Participants) == 0
		return nil
	})
	if err != nil {
		problem(w, status(err), err.Error())
		return
	}
	if destroy {
		room := m.LiveKitRoom
		a.meetings.Delete(id)
		a.hub.Broadcast(id, ws.Event{Type: "meeting.ended"})
		w.WriteHeader(204)
		a.deleteLiveKitRoom(room)
		return
	}
	a.hub.Broadcast(id, ws.Event{Type: "meeting.updated", Payload: broadcastMeeting(m)})
	w.WriteHeader(204)
}
func (a *API) remove(w http.ResponseWriter, r *http.Request) {
	host := auth.User(r)
	id := chi.URLParam(r, "id")
	var body struct {
		UserID string `json:"userId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	m, err := a.meetings.Update(id, func(m *models.Meeting) error {
		if m.HostID != host.ID {
			return errors.New("host only")
		}
		if body.UserID == host.ID {
			return errors.New("cannot remove host")
		}
		delete(m.Participants, body.UserID)
		return nil
	})
	if err != nil {
		problem(w, status(err), err.Error())
		return
	}
	a.hub.Broadcast(id, ws.Event{Type: "participant.removed", UserID: body.UserID, Payload: broadcastMeeting(m)})
	w.WriteHeader(204)
}
func (a *API) mute(w http.ResponseWriter, r *http.Request) {
	host := auth.User(r)
	id := chi.URLParam(r, "id")
	var body struct {
		UserID string `json:"userId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	m, err := a.meetings.Update(id, func(m *models.Meeting) error {
		if m.HostID != host.ID {
			return errors.New("host only")
		}
		p := m.Participants[body.UserID]
		if p == nil {
			return errors.New("participant not found")
		}
		p.MicEnabled = false
		return nil
	})
	if err != nil {
		problem(w, status(err), err.Error())
		return
	}
	a.hub.Broadcast(id, ws.Event{Type: "participant.muted", UserID: body.UserID, Payload: broadcastMeeting(m)})
	w.WriteHeader(204)
}
func (a *API) chat(w http.ResponseWriter, r *http.Request) {
	user := auth.User(r)
	id := chi.URLParam(r, "id")
	var body struct {
		Text        string `json:"text"`
		RecipientID string `json:"recipientId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	body.Text = strings.TrimSpace(body.Text)
	body.RecipientID = strings.TrimSpace(body.RecipientID)
	if body.Text == "" || len(body.Text) > 1000 {
		problem(w, 400, "message must be 1-1000 characters")
		return
	}
	msg := models.ChatMessage{ID: uuid.NewString(), UserID: user.ID, DisplayName: user.DisplayName, Text: body.Text, SentAt: time.Now().UTC()}
	_, err := a.meetings.Update(id, func(m *models.Meeting) error {
		if m.Participants[user.ID] == nil {
			return errors.New("not in meeting")
		}
		if body.RecipientID != "" {
			if body.RecipientID == user.ID {
				return errors.New("cannot message yourself")
			}
			recipient := m.Participants[body.RecipientID]
			if recipient == nil {
				return errors.New("recipient not in meeting")
			}
			msg.RecipientID = body.RecipientID
			msg.RecipientName = recipient.DisplayName
		}
		m.Chat = append(m.Chat, msg)
		return nil
	})
	if err != nil {
		problem(w, status(err), err.Error())
		return
	}
	event := ws.Event{Type: "chat.message", Payload: msg}
	if msg.RecipientID == "" {
		a.hub.Broadcast(id, event)
	} else {
		a.hub.SendTo(id, user.ID, event)
		a.hub.SendTo(id, msg.RecipientID, event)
	}
	write(w, 201, msg)
}
func (a *API) media(w http.ResponseWriter, r *http.Request) {
	user := auth.User(r)
	id := chi.URLParam(r, "id")
	var body struct{ Mic, Camera, Screen *bool }
	_ = json.NewDecoder(r.Body).Decode(&body)
	m, err := a.meetings.Update(id, func(m *models.Meeting) error {
		p := m.Participants[user.ID]
		if p == nil {
			return errors.New("not in meeting")
		}
		if body.Mic != nil {
			p.MicEnabled = *body.Mic
		}
		if body.Camera != nil {
			p.CameraEnabled = *body.Camera
		}
		if body.Screen != nil {
			p.ScreenSharing = *body.Screen
		}
		return nil
	})
	if err != nil {
		problem(w, status(err), err.Error())
		return
	}
	a.hub.Broadcast(id, ws.Event{Type: "participant.media", UserID: user.ID, Payload: broadcastMeeting(m)})
	w.WriteHeader(204)
}
func (a *API) end(w http.ResponseWriter, r *http.Request) {
	host := auth.User(r)
	id := chi.URLParam(r, "id")
	var room string
	_, err := a.meetings.Update(id, func(m *models.Meeting) error {
		if m.HostID != host.ID {
			return errors.New("host only")
		}
		m.State = models.MeetingEnding
		room = m.LiveKitRoom
		return nil
	})
	if err != nil {
		problem(w, status(err), err.Error())
		return
	}
	a.hub.Broadcast(id, ws.Event{Type: "meeting.ended"})
	a.meetings.Delete(id)
	w.WriteHeader(204)
	a.deleteLiveKitRoom(room)
}
func (a *API) deleteLiveKitRoom(room string) {
	if room == "" {
		return
	}
	go func() {
		// LiveKit.DeleteRoom ignores this parent and applies its own timeout/retry.
		if err := a.livekit.DeleteRoom(context.Background(), room); err != nil {
			slog.Error("livekit room cleanup failed after retries", "room", room, "error", err)
		}
	}()
}
func participant(u models.User, host bool) *models.Participant {
	return &models.Participant{UserID: u.ID, DisplayName: u.DisplayName, Avatar: u.Avatar, IsHost: host, JoinedAt: time.Now().UTC(), MicEnabled: true, CameraEnabled: true}
}
func publicMeeting(m *models.Meeting, userID string) map[string]any {
	return meetingPayload(m, visibleChat(m, userID), userID)
}

// broadcastMeeting is safe for room-wide WebSocket fan-out: only public chat, no DMs.
func broadcastMeeting(m *models.Meeting) map[string]any {
	return meetingPayload(m, publicChatOnly(m), "")
}

func meetingPayload(m *models.Meeting, chat []models.ChatMessage, viewerID string) map[string]any {
	return map[string]any{
		"id": m.ID, "hostId": m.HostID, "createdAt": m.CreatedAt,
		"participants": m.Participants, "waitingRoom": m.WaitingRoom,
		"chat": chat, "state": m.State, "isHost": m.HostID == viewerID,
	}
}

func publicChatOnly(m *models.Meeting) []models.ChatMessage {
	out := make([]models.ChatMessage, 0, len(m.Chat))
	for _, msg := range m.Chat {
		if msg.RecipientID == "" {
			out = append(out, msg)
		}
	}
	return out
}

func visibleChat(m *models.Meeting, viewerID string) []models.ChatMessage {
	admitted := m.Participants[viewerID] != nil
	out := make([]models.ChatMessage, 0, len(m.Chat))
	for _, msg := range m.Chat {
		if msg.RecipientID == "" {
			out = append(out, msg)
			continue
		}
		if !admitted {
			continue
		}
		if msg.UserID == viewerID || msg.RecipientID == viewerID {
			out = append(out, msg)
		}
	}
	return out
}
func status(err error) int {
	if errors.Is(err, meeting.ErrNotFound) {
		return 404
	}
	if err.Error() == "host only" {
		return 403
	}
	return 400
}
func problem(w http.ResponseWriter, code int, message string) {
	write(w, code, map[string]string{"error": message})
}
func write(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
