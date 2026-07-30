package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/instantmeet/instantmeet/backend/internal/auth"
	"github.com/instantmeet/instantmeet/backend/internal/config"
	"github.com/instantmeet/instantmeet/backend/internal/models"
)

func TestLoginErrorUsesBrowserLanguage(t *testing.T) {
	svc := auth.New(config.Config{}, nil)
	tests := []struct {
		name     string
		language string
		want     string
	}{
		{name: "English fallback", language: "en-US,en;q=0.9", want: "Google login is not configured."},
		{name: "Turkish", language: "tr-TR,tr;q=0.9,en;q=0.8", want: "Google ile giriş yapılandırılmamış."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/login/google", nil)
			req.Header.Set("Accept-Language", tt.language)
			res := httptest.NewRecorder()

			svc.Login(res, req)

			if res.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d", res.Code)
			}
			if got := strings.TrimSpace(res.Body.String()); got != tt.want {
				t.Fatalf("body = %q, want %q", got, tt.want)
			}
		})
	}
}

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
