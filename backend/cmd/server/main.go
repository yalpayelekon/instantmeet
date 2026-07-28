package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/instantmeet/instantmeet/backend/internal/api"
	"github.com/instantmeet/instantmeet/backend/internal/auth"
	"github.com/instantmeet/instantmeet/backend/internal/config"
	"github.com/instantmeet/instantmeet/backend/internal/meeting"
	"github.com/instantmeet/instantmeet/backend/internal/services"
	ws "github.com/instantmeet/instantmeet/backend/internal/websocket"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	authService := auth.New(cfg)
	hub := ws.NewHub(authService)
	handler := api.New(meeting.NewStore(), hub, services.NewLiveKit(cfg))

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Use(requestLogger)
	r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{cfg.FrontendURL}, AllowedMethods: []string{"GET", "POST", "OPTIONS"}, AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"}, AllowCredentials: true, MaxAge: 300}))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	r.Get("/api/login/google", authService.Login)
	r.Post("/api/login/google", authService.Login)
	r.Get("/api/auth/google/callback", authService.Callback)
	r.Get("/api/login/dev", authService.DevLogin)
	r.Post("/api/logout", authService.Logout)
	r.With(authService.Middleware).Mount("/api", handler.Routes())
	r.Handle("/ws", hub)

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: r, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		slog.Info("server started", "addr", cfg.HTTPAddr, "env", cfg.Environment)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("shutdown failed", "error", err)
	}
	slog.Info("server stopped")
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String(), "request_id", middleware.GetReqID(r.Context()))
	})
}
