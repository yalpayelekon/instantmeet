package websocket_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/instantmeet/instantmeet/backend/internal/auth"
	"github.com/instantmeet/instantmeet/backend/internal/config"
	"github.com/instantmeet/instantmeet/backend/internal/meeting"
	"github.com/instantmeet/instantmeet/backend/internal/models"
	ws "github.com/instantmeet/instantmeet/backend/internal/websocket"
)

func TestServeHTTPRejectsNonMember(t *testing.T) {
	cfg := config.Config{JWTSecret: "test-secret-at-least-32-characters!!", FrontendURL: "http://localhost:5173"}
	svc := auth.New(cfg, nil)
	store := meeting.NewStore()
	m := store.Create(models.User{ID: "host"})
	hub := ws.NewHub(svc, store, cfg.FrontendURL)

	token, err := svc.Sign(models.User{ID: "outsider", DisplayName: "Out"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ws?meetingId="+m.ID, nil)
	req.Header.Set("Origin", cfg.FrontendURL)
	req.AddCookie(&http.Cookie{Name: "instantmeet_token", Value: token})
	rr := httptest.NewRecorder()
	hub.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestServeHTTPRejectsBadOrigin(t *testing.T) {
	cfg := config.Config{JWTSecret: "test-secret-at-least-32-characters!!", FrontendURL: "http://localhost:5173"}
	svc := auth.New(cfg, nil)
	store := meeting.NewStore()
	host := models.User{ID: "host", DisplayName: "Host"}
	m := store.Create(host)
	_, _ = store.Update(m.ID, func(meeting *models.Meeting) error {
		meeting.Participants[host.ID] = &models.Participant{UserID: host.ID, DisplayName: host.DisplayName, IsHost: true}
		return nil
	})
	hub := ws.NewHub(svc, store, cfg.FrontendURL)

	token, err := svc.Sign(host, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ws?meetingId="+m.ID, nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-Websocket-Version", "13")
	req.Header.Set("Sec-Websocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.AddCookie(&http.Cookie{Name: "instantmeet_token", Value: token})
	rr := httptest.NewRecorder()
	hub.ServeHTTP(rr, req)
	if rr.Code == http.StatusSwitchingProtocols {
		t.Fatal("expected origin rejection")
	}
	if rr.Code != http.StatusForbidden && !strings.Contains(rr.Body.String(), "origin") && rr.Code != http.StatusBadRequest {
		// gorilla returns 403 when CheckOrigin fails
		if rr.Code != http.StatusForbidden {
			t.Fatalf("unexpected status %d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func TestServeHTTPRejectsUnauthorized(t *testing.T) {
	cfg := config.Config{JWTSecret: "test-secret-at-least-32-characters!!", FrontendURL: "http://localhost"}
	hub := ws.NewHub(auth.New(cfg, nil), meeting.NewStore(), cfg.FrontendURL)
	req := httptest.NewRequest(http.MethodGet, "/ws?meetingId=abc", nil)
	req.Header.Set("Origin", cfg.FrontendURL)
	rr := httptest.NewRecorder()
	hub.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rr.Code)
	}
}
