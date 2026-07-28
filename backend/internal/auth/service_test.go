package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/instantmeet/instantmeet/backend/internal/auth"
	"github.com/instantmeet/instantmeet/backend/internal/config"
	"github.com/instantmeet/instantmeet/backend/internal/models"
)

func TestUserFromRequestIgnoresQueryToken(t *testing.T) {
	svc := auth.New(config.Config{JWTSecret: "test-secret-at-least-32-characters!!"}, nil)
	user := models.User{ID: "u1", Email: "u@example.com", DisplayName: "U"}
	token, err := svc.Sign(user, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me?token="+token, nil)
	if _, err := svc.UserFromRequest(req); err == nil {
		t.Fatal("expected query token to be rejected")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "instantmeet_token", Value: token})
	got, err := svc.UserFromRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "u1" {
		t.Fatalf("got %#v", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	got, err = svc.UserFromRequest(req)
	if err != nil || got.ID != "u1" {
		t.Fatalf("bearer failed: %#v %v", got, err)
	}
}
