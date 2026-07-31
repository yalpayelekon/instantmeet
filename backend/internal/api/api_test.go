package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/instantmeet/instantmeet/backend/internal/api"
	"github.com/instantmeet/instantmeet/backend/internal/auth"
	"github.com/instantmeet/instantmeet/backend/internal/config"
	"github.com/instantmeet/instantmeet/backend/internal/meeting"
	"github.com/instantmeet/instantmeet/backend/internal/models"
	ws "github.com/instantmeet/instantmeet/backend/internal/websocket"
)

type failLiveKit struct{}

func (failLiveKit) Token(string, models.User, bool) (string, error) {
	return "", errors.New("mint failed")
}
func (failLiveKit) PublicURL() string { return "ws://localhost:7880" }
func (failLiveKit) DeleteRoom(context.Context, string) error { return nil }

func TestJoinReturnsBadGatewayWhenLiveKitFails(t *testing.T) {
	cfg := config.Config{JWTSecret: "test-secret-at-least-32-characters!!", FrontendURL: "http://localhost"}
	svc := auth.New(cfg, nil)
	store := meeting.NewStore()
	hub := ws.NewHub(svc, store, cfg.FrontendURL)
	handler := api.New(store, hub, failLiveKit{})

	host := models.User{ID: "host", DisplayName: "Host", Email: "h@example.com"}
	m := store.Create(host)

	r := chi.NewRouter()
	r.Use(svc.Middleware)
	r.Mount("/", handler.Routes())

	token, err := svc.Sign(host, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/meetings/"+m.ID+"/join", nil)
	req.AddCookie(&http.Cookie{Name: "instantmeet_token", Value: token})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body["error"] == "" {
		t.Fatalf("expected error payload, got %#v", body)
	}
}

type trackingLiveKit struct {
	deletes atomic.Int32
	lastRoom atomic.Value
}

func (t *trackingLiveKit) Token(string, models.User, bool) (string, error) {
	return "token", nil
}
func (t *trackingLiveKit) PublicURL() string { return "ws://localhost:7880" }
func (t *trackingLiveKit) DeleteRoom(_ context.Context, room string) error {
	t.deletes.Add(1)
	t.lastRoom.Store(room)
	return nil
}

func TestEndDeletesLiveKitRoom(t *testing.T) {
	cfg := config.Config{JWTSecret: "test-secret-at-least-32-characters!!", FrontendURL: "http://localhost"}
	svc := auth.New(cfg, nil)
	store := meeting.NewStore()
	hub := ws.NewHub(svc, store, cfg.FrontendURL)
	lk := &trackingLiveKit{}
	handler := api.New(store, hub, lk)

	host := models.User{ID: "host", DisplayName: "Host", Email: "h@example.com"}
	m := store.Create(host)

	r := chi.NewRouter()
	r.Use(svc.Middleware)
	r.Mount("/", handler.Routes())

	token, err := svc.Sign(host, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/meetings/"+m.ID+"/join", nil)
	req.AddCookie(&http.Cookie{Name: "instantmeet_token", Value: token})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("join status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/meetings/"+m.ID+"/end", nil)
	req.AddCookie(&http.Cookie{Name: "instantmeet_token", Value: token})
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("end status=%d body=%s", rr.Code, rr.Body.String())
	}
	deadline := time.Now().Add(2 * time.Second)
	for lk.deletes.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if lk.deletes.Load() != 1 {
		t.Fatalf("expected DeleteRoom once, got %d", lk.deletes.Load())
	}
	if got, _ := lk.lastRoom.Load().(string); got != m.LiveKitRoom {
		t.Fatalf("DeleteRoom room=%q want %q", got, m.LiveKitRoom)
	}
	if _, ok := store.Get(m.ID); ok {
		t.Fatal("meeting should be deleted")
	}
}

type okLiveKit struct{}

func (okLiveKit) Token(string, models.User, bool) (string, error) { return "token", nil }
func (okLiveKit) PublicURL() string                               { return "ws://localhost:7880" }
func (okLiveKit) DeleteRoom(context.Context, string) error        { return nil }

func chatTestEnv(t *testing.T) (*meeting.Store, http.Handler, *auth.Service) {
	t.Helper()
	cfg := config.Config{JWTSecret: "test-secret-at-least-32-characters!!", FrontendURL: "http://localhost"}
	svc := auth.New(cfg, nil)
	store := meeting.NewStore()
	hub := ws.NewHub(svc, store, cfg.FrontendURL)
	handler := api.New(store, hub, okLiveKit{})
	r := chi.NewRouter()
	r.Use(svc.Middleware)
	r.Mount("/", handler.Routes())
	return store, r, svc
}

func authToken(t *testing.T, svc *auth.Service, user models.User) string {
	t.Helper()
	token, err := svc.Sign(user, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func doAuthed(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{Name: "instantmeet_token", Value: token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestChatPrivateMessageVisibility(t *testing.T) {
	store, h, svc := chatTestEnv(t)
	host := models.User{ID: "host", DisplayName: "Host", Email: "h@example.com"}
	guest := models.User{ID: "guest", DisplayName: "Guest", Email: "g@example.com"}
	other := models.User{ID: "other", DisplayName: "Other", Email: "o@example.com"}
	hostTok, guestTok, otherTok := authToken(t, svc, host), authToken(t, svc, guest), authToken(t, svc, other)

	m := store.Create(host)
	if rr := doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/join", hostTok, ""); rr.Code != http.StatusOK {
		t.Fatalf("host join: %d %s", rr.Code, rr.Body.String())
	}
	if rr := doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/join", guestTok, ""); rr.Code != http.StatusOK {
		t.Fatalf("guest join: %d %s", rr.Code, rr.Body.String())
	}
	if rr := doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/admit", hostTok, `{"userId":"guest"}`); rr.Code != http.StatusOK {
		t.Fatalf("admit guest: %d %s", rr.Code, rr.Body.String())
	}
	if rr := doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/join", otherTok, ""); rr.Code != http.StatusOK {
		t.Fatalf("other join: %d %s", rr.Code, rr.Body.String())
	}
	if rr := doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/admit", hostTok, `{"userId":"other"}`); rr.Code != http.StatusOK {
		t.Fatalf("admit other: %d %s", rr.Code, rr.Body.String())
	}

	rr := doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/chat", hostTok, `{"text":"secret","recipientId":"guest"}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("private chat: %d %s", rr.Code, rr.Body.String())
	}
	var msg map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&msg)
	if msg["recipientId"] != "guest" || msg["recipientName"] != "Guest" {
		t.Fatalf("expected recipient on message, got %#v", msg)
	}

	rr = doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/chat", hostTok, `{"text":"hello all"}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("public chat: %d %s", rr.Code, rr.Body.String())
	}

	chatIDs := func(token string) []string {
		t.Helper()
		get := doAuthed(t, h, http.MethodGet, "/meetings/"+m.ID, token, "")
		if get.Code != http.StatusOK {
			t.Fatalf("get: %d %s", get.Code, get.Body.String())
		}
		var body struct {
			Chat []struct {
				Text        string `json:"text"`
				RecipientID string `json:"recipientId"`
			} `json:"chat"`
		}
		_ = json.NewDecoder(get.Body).Decode(&body)
		texts := make([]string, len(body.Chat))
		for i, c := range body.Chat {
			texts[i] = c.Text
			if c.Text == "secret" && c.RecipientID == "" {
				t.Fatal("private message missing recipientId in response")
			}
		}
		return texts
	}

	hostChat := chatIDs(hostTok)
	guestChat := chatIDs(guestTok)
	otherChat := chatIDs(otherTok)

	if !contains(hostChat, "secret") || !contains(hostChat, "hello all") {
		t.Fatalf("host chat=%v", hostChat)
	}
	if !contains(guestChat, "secret") || !contains(guestChat, "hello all") {
		t.Fatalf("guest chat=%v", guestChat)
	}
	if contains(otherChat, "secret") {
		t.Fatalf("other should not see DM, got %v", otherChat)
	}
	if !contains(otherChat, "hello all") {
		t.Fatalf("other should see public chat, got %v", otherChat)
	}
}

func TestChatPrivateValidation(t *testing.T) {
	store, h, svc := chatTestEnv(t)
	host := models.User{ID: "host", DisplayName: "Host", Email: "h@example.com"}
	guest := models.User{ID: "guest", DisplayName: "Guest", Email: "g@example.com"}
	hostTok, guestTok := authToken(t, svc, host), authToken(t, svc, guest)
	m := store.Create(host)
	doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/join", hostTok, "")
	doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/join", guestTok, "")

	// Guest still waiting — cannot send; host cannot DM waiting user as recipient not in participants.
	if rr := doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/chat", hostTok, `{"text":"hi","recipientId":"guest"}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("DM to waiting: status=%d body=%s", rr.Code, rr.Body.String())
	}
	doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/admit", hostTok, `{"userId":"guest"}`)

	if rr := doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/chat", hostTok, `{"text":"hi","recipientId":"host"}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("self DM: status=%d", rr.Code)
	}
	if rr := doAuthed(t, h, http.MethodPost, "/meetings/"+m.ID+"/chat", hostTok, `{"text":"hi","recipientId":"missing"}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("missing recipient: status=%d", rr.Code)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
