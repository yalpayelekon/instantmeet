package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type stubCollector struct{}

func (stubCollector) MeetingStats() (int, int, int) { return 0, 0, 0 }
func (stubCollector) WebSocketConnections() int     { return 0 }

func TestRoutePathUsesChiPattern(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/meetings/abc-def-ghi/join", nil)
	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/api/meetings/{id}/join"}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	if got := routePath(req); got != "/api/meetings/{id}/join" {
		t.Fatalf("routePath = %q, want chi pattern", got)
	}
}

func TestRoutePathFallbacks(t *testing.T) {
	ws := httptest.NewRequest(http.MethodGet, "/ws", nil)
	if got := routePath(ws); got != "/ws" {
		t.Fatalf("ws path = %q", got)
	}
	other := httptest.NewRequest(http.MethodGet, "/nope", nil)
	if got := routePath(other); got != "unmatched" {
		t.Fatalf("unknown path = %q", got)
	}
}

func TestMiddlewareRecordsChiRoutePattern(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	m := New(stubCollector{})
	r := chi.NewRouter()
	r.Use(m.Middleware)
	r.Get("/api/meetings/{id}/join", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/meetings/abc-def-ghi/join", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	if err := testutil.CollectAndCompare(m.requests, strings.NewReader(`
# HELP instantmeet_http_requests_total Total HTTP requests excluding health and metrics.
# TYPE instantmeet_http_requests_total counter
instantmeet_http_requests_total{method="GET",path="/api/meetings/{id}/join",status="204"} 1
`), "instantmeet_http_requests_total"); err != nil {
		t.Fatal(err)
	}
}
