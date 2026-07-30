package config

import (
	"testing"
)

func TestLoadProductionRequiresGoogle(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "production-jwt-secret-at-least-32-chars")
	t.Setenv("LIVEKIT_API_KEY", "prodkey")
	t.Setenv("LIVEKIT_API_SECRET", "production-livekit-secret-value")
	t.Setenv("FRONTEND_URL", "https://toplanti.online")
	t.Setenv("LIVEKIT_PUBLIC_URL", "wss://livekit.toplanti.online")
	t.Setenv("GOOGLE_REDIRECT_URL", "https://toplanti.online/api/auth/google/callback")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	t.Setenv("DEV_AUTH_ENABLED", "false")

	if _, err := Load(); err == nil {
		t.Fatal("expected error when Google OAuth is empty in production")
	}
}

func TestLoadProductionSucceedsWithGoogle(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "production-jwt-secret-at-least-32-chars")
	t.Setenv("LIVEKIT_API_KEY", "prodkey")
	t.Setenv("LIVEKIT_API_SECRET", "production-livekit-secret-value")
	t.Setenv("FRONTEND_URL", "https://toplanti.online")
	t.Setenv("LIVEKIT_PUBLIC_URL", "wss://livekit.toplanti.online")
	t.Setenv("GOOGLE_REDIRECT_URL", "https://toplanti.online/api/auth/google/callback")
	t.Setenv("GOOGLE_CLIENT_ID", "client.apps.googleusercontent.com")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret-value")
	t.Setenv("DEV_AUTH_ENABLED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GoogleClientID == "" {
		t.Fatal("expected Google client id")
	}
}

func TestLoadDevelopmentAllowsEmptyGoogle(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", "development-only-change-this-secret")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	t.Setenv("DEV_AUTH_ENABLED", "true")

	if _, err := Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
