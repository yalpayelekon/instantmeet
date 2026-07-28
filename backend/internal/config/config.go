package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Environment, HTTPAddr, FrontendURL, DatabaseURL, RedisURL        string
	JWTSecret, GoogleClientID, GoogleClientSecret, GoogleRedirectURL string
	LiveKitURL, LiveKitPublicURL, LiveKitAPIKey, LiveKitAPISecret    string
	DevAuthEnabled                                                   bool
}

func Load() (Config, error) {
	c := Config{
		Environment:        get("APP_ENV", "development"),
		HTTPAddr:           get("HTTP_ADDR", ":8080"),
		FrontendURL:        get("FRONTEND_URL", "http://localhost"),
		DatabaseURL:        get("DATABASE_URL", "postgres://instantmeet:instantmeet@localhost:5432/instantmeet?sslmode=disable"),
		RedisURL:           get("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:          get("JWT_SECRET", "development-only-change-this-secret"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  get("GOOGLE_REDIRECT_URL", "http://localhost/api/auth/google/callback"),
		LiveKitURL:         get("LIVEKIT_URL", "ws://localhost:7880"),
		LiveKitPublicURL:   get("LIVEKIT_PUBLIC_URL", "ws://localhost:7880"),
		LiveKitAPIKey:      get("LIVEKIT_API_KEY", "devkey"),
		LiveKitAPISecret:   get("LIVEKIT_API_SECRET", "secret"),
	}
	c.DevAuthEnabled, _ = strconv.ParseBool(get("DEV_AUTH_ENABLED", "false"))
	if c.Environment == "production" && len(c.JWTSecret) < 32 {
		return c, fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	return c, nil
}

func get(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
