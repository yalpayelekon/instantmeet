package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/instantmeet/instantmeet/backend/internal/api"
	"github.com/instantmeet/instantmeet/backend/internal/auth"
	"github.com/instantmeet/instantmeet/backend/internal/config"
	"github.com/instantmeet/instantmeet/backend/internal/db"
	"github.com/instantmeet/instantmeet/backend/internal/meeting"
	"github.com/instantmeet/instantmeet/backend/internal/services"
	ws "github.com/instantmeet/instantmeet/backend/internal/websocket"
)

type metrics struct {
	requests atomic.Int64
	errors   atomic.Int64
	latency  atomic.Int64 // nanoseconds total for average
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	authService := auth.New(cfg, pool)
	store := meeting.NewStore()
	hub := ws.NewHub(authService, store, cfg.FrontendURL)
	handler := api.New(store, hub, services.NewLiveKit(cfg))
	m := &metrics{}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Use(requestLogger(m))
	r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{cfg.FrontendURL}, AllowedMethods: []string{"GET", "POST", "OPTIONS"}, AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"}, AllowCredentials: true, MaxAge: 300}))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	r.Get("/metrics", m.serve)
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
	}
	slog.Info("server stopped")
}

func requestLogger(m *metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			duration := time.Since(start)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			m.requests.Add(1)
			m.latency.Add(duration.Nanoseconds())
			if status >= 500 {
				m.errors.Add(1)
			}
			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", duration.Milliseconds(),
				"bytes", ww.BytesWritten(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}

func (m *metrics) serve(w http.ResponseWriter, _ *http.Request) {
	total := m.requests.Load()
	errs := m.errors.Load()
	latency := m.latency.Load()
	var avgMs float64
	if total > 0 {
		avgMs = float64(latency) / float64(total) / 1e6
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	var b strings.Builder
	b.WriteString("# HELP instantmeet_http_requests_total Total HTTP requests excluding health and metrics.\n")
	b.WriteString("# TYPE instantmeet_http_requests_total counter\n")
	b.WriteString("instantmeet_http_requests_total ")
	b.WriteString(strconv.FormatInt(total, 10))
	b.WriteString("\n")
	b.WriteString("# HELP instantmeet_http_errors_total HTTP responses with status >= 500.\n")
	b.WriteString("# TYPE instantmeet_http_errors_total counter\n")
	b.WriteString("instantmeet_http_errors_total ")
	b.WriteString(strconv.FormatInt(errs, 10))
	b.WriteString("\n")
	b.WriteString("# HELP instantmeet_http_request_duration_avg_ms Average request duration in milliseconds.\n")
	b.WriteString("# TYPE instantmeet_http_request_duration_avg_ms gauge\n")
	b.WriteString("instantmeet_http_request_duration_avg_ms ")
	b.WriteString(strconv.FormatFloat(avgMs, 'f', 3, 64))
	b.WriteString("\n")
	_, _ = w.Write([]byte(b.String()))
}
