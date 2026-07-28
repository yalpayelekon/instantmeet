package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
