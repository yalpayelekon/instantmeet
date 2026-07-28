package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/instantmeet/instantmeet/backend/internal/config"
	"github.com/instantmeet/instantmeet/backend/internal/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type contextKey string

const userKey contextKey = "user"

type Service struct {
	cfg    config.Config
	oauth  *oauth2.Config
	states sync.Map
}

type claims struct {
	Email, Name, Avatar string
	jwt.RegisteredClaims
}

func New(cfg config.Config) *Service {
	return &Service{cfg: cfg, oauth: &oauth2.Config{
		ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret,
		RedirectURL: cfg.GoogleRedirectURL,
		Scopes:      []string{"openid", "email", "profile"},
		Endpoint:    google.Endpoint,
	}}
}

func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	if s.oauth.ClientID == "" {
		http.Error(w, `{"error":"Google OAuth is not configured"}`, http.StatusServiceUnavailable)
		return
	}
	state := randomState()
	s.states.Store(state, time.Now().Add(10*time.Minute))
	http.Redirect(w, r, s.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline), http.StatusTemporaryRedirect)
}

func (s *Service) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	expires, ok := s.states.LoadAndDelete(state)
	if !ok || time.Now().After(expires.(time.Time)) {
		http.Error(w, "invalid OAuth state", 400)
		return
	}
	tok, err := s.oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "OAuth exchange failed", 401)
		return
	}
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		http.Error(w, "profile lookup failed", 401)
		return
	}
	defer res.Body.Close()
	var profile struct{ Sub, Email, Name, Picture string }
	if json.NewDecoder(res.Body).Decode(&profile) != nil {
		http.Error(w, "invalid profile", 401)
		return
	}
	user := models.User{ID: profile.Sub, GoogleID: profile.Sub, Email: profile.Email, DisplayName: profile.Name, Avatar: profile.Picture}
	s.completeLogin(w, r, user)
}

func (s *Service) DevLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.DevAuthEnabled {
		http.NotFound(w, r)
		return
	}
	user := models.User{ID: "dev-" + uuid.NewString(), Email: "demo@instantmeet.local", DisplayName: "Demo User"}
	s.completeLogin(w, r, user)
}

func (s *Service) completeLogin(w http.ResponseWriter, r *http.Request, user models.User) {
	token, _ := s.Sign(user, 12*time.Hour)
	http.SetCookie(w, &http.Cookie{Name: "instantmeet_token", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.cfg.Environment == "production", MaxAge: 43200})
	http.Redirect(w, r, s.cfg.FrontendURL, http.StatusTemporaryRedirect)
}

func (s *Service) Logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "instantmeet_token", Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) Sign(user models.User, ttl time.Duration) (string, error) {
	now := time.Now()
	c := claims{Email: user.Email, Name: user.DisplayName, Avatar: user.Avatar, RegisteredClaims: jwt.RegisteredClaims{
		Subject: user.ID, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(ttl)), Issuer: "instantmeet",
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(s.cfg.JWTSecret))
}

func (s *Service) UserFromRequest(r *http.Request) (models.User, error) {
	raw := ""
	if cookie, err := r.Cookie("instantmeet_token"); err == nil {
		raw = cookie.Value
	}
	if raw == "" && strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		raw = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if raw == "" {
		if q, err := url.QueryUnescape(r.URL.Query().Get("token")); err == nil {
			raw = q
		}
	}
	var c claims
	t, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !t.Valid {
		return models.User{}, errors.New("unauthorized")
	}
	return models.User{ID: c.Subject, Email: c.Email, DisplayName: c.Name, Avatar: c.Avatar}, nil
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.UserFromRequest(r)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, user)))
	})
}

func User(r *http.Request) models.User {
	user, _ := r.Context().Value(userKey).(models.User)
	return user
}

func randomState() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
