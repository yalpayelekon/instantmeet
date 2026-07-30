package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	if err := c.validate(); err != nil {
		return c, err
	}
	return c, nil
}

func (c Config) validate() error {
	if c.Environment != "production" {
		return nil
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	if c.JWTSecret == "development-only-change-this-secret" || c.JWTSecret == "replace-with-at-least-32-random-characters" {
		return fmt.Errorf("JWT_SECRET must be replaced with a unique production secret")
	}
	if c.LiveKitAPIKey == "" || c.LiveKitAPIKey == "devkey" {
		return fmt.Errorf("LIVEKIT_API_KEY must be set to a non-default value in production")
	}
	if c.LiveKitAPISecret == "" || c.LiveKitAPISecret == "secret" {
		return fmt.Errorf("LIVEKIT_API_SECRET must be set to a non-default value in production")
	}
	if !c.DevAuthEnabled {
		if strings.TrimSpace(c.GoogleClientID) == "" {
			return fmt.Errorf("GOOGLE_CLIENT_ID is required in production")
		}
		if strings.TrimSpace(c.GoogleClientSecret) == "" {
			return fmt.Errorf("GOOGLE_CLIENT_SECRET is required in production")
		}
	}
	if !strings.HasPrefix(c.GoogleRedirectURL, "https://") {
		return fmt.Errorf("GOOGLE_REDIRECT_URL must use https in production")
	}
	if !strings.HasPrefix(c.FrontendURL, "https://") {
		return fmt.Errorf("FRONTEND_URL must use https in production")
	}
	if !strings.HasPrefix(c.LiveKitPublicURL, "wss://") {
		return fmt.Errorf("LIVEKIT_PUBLIC_URL must use wss in production")
	}
	return nil
}

func get(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
