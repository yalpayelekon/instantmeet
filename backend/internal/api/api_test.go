package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
