package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/instantmeet/instantmeet/backend/internal/config"
	"github.com/instantmeet/instantmeet/backend/internal/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type contextKey string

const userKey contextKey = "user"

// UserRepository persists authenticated users (Postgres in production).
type UserRepository interface {
	UpsertUser(ctx context.Context, user models.User) error
}

type Service struct {
	cfg    config.Config
	oauth  *oauth2.Config
	states sync.Map
	users  UserRepository
}

type claims struct {
	Email, Name, Avatar string
	jwt.RegisteredClaims
}

func New(cfg config.Config, users UserRepository) *Service {
	return &Service{cfg: cfg, users: users, oauth: &oauth2.Config{
		ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret,
		RedirectURL: cfg.GoogleRedirectURL,
		Scopes:      []string{"openid", "email", "profile"},
		Endpoint:    google.Endpoint,
	}}
}

func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	if s.oauth.ClientID == "" {
		oauthError(w, r, "not_configured", http.StatusServiceUnavailable)
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
		oauthError(w, r, "invalid_state", http.StatusBadRequest)
		return
	}
	tok, err := s.oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		oauthError(w, r, "exchange_failed", http.StatusUnauthorized)
		return
	}
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		oauthError(w, r, "profile_failed", http.StatusUnauthorized)
		return
	}
	defer res.Body.Close()
	var profile struct{ Sub, Email, Name, Picture string }
	if json.NewDecoder(res.Body).Decode(&profile) != nil {
		oauthError(w, r, "invalid_profile", http.StatusUnauthorized)
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
	user := models.User{ID: "dev-local", GoogleID: "dev-local", Email: "demo@instantmeet.local", DisplayName: "Demo User"}
	s.completeLogin(w, r, user)
}

func (s *Service) completeLogin(w http.ResponseWriter, r *http.Request, user models.User) {
	if s.users != nil {
		if err := s.users.UpsertUser(r.Context(), user); err != nil {
			slog.Error("persist user failed", "error", err, "user_id", user.ID)
			oauthError(w, r, "persist_failed", http.StatusInternalServerError)
			return
		}
	}
	token, err := s.Sign(user, 12*time.Hour)
	if err != nil {
		oauthError(w, r, "session_failed", http.StatusInternalServerError)
		return
	}
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
		return models.User{}, errors.New("unauthorized")
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

var oauthErrors = map[string][2]string{
	"not_configured": {"Google login is not configured.", "Google ile giriş yapılandırılmamış."},
	"invalid_state":  {"The login request expired or is invalid. Please try again.", "Giriş isteğinin süresi dolmuş veya istek geçersiz. Lütfen yeniden deneyin."},
	"exchange_failed": {"Google login could not be completed. Please try again.",
		"Google ile giriş tamamlanamadı. Lütfen yeniden deneyin."},
	"profile_failed":  {"Your Google profile could not be retrieved.", "Google profiliniz alınamadı."},
	"invalid_profile": {"Google returned an invalid profile.", "Google geçersiz bir profil döndürdü."},
	"persist_failed":  {"Your account could not be saved.", "Hesabınız kaydedilemedi."},
	"session_failed":  {"Your session could not be created.", "Oturumunuz oluşturulamadı."},
}

func oauthError(w http.ResponseWriter, r *http.Request, key string, status int) {
	message := oauthErrors[key][0]
	firstLanguage := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Accept-Language"), ",")[0]))
	if firstLanguage == "tr" || strings.HasPrefix(firstLanguage, "tr-") {
		message = oauthErrors[key][1]
	}
	http.Error(w, message, status)
}
