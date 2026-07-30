package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector supplies live business gauges without high-cardinality labels.
type Collector interface {
	MeetingStats() (meetings, participants, waiting int)
	WebSocketConnections() int
}

type Metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func New(c Collector) *Metrics {
	m := &Metrics{
		requests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "instantmeet_http_requests_total",
			Help: "Total HTTP requests excluding health and metrics.",
		}, []string{"method", "path", "status"}),
		duration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "instantmeet_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "path", "status"}),
	}

	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "instantmeet_meetings_active",
		Help: "In-memory meetings currently stored.",
	}, func() float64 {
		n, _, _ := c.MeetingStats()
		return float64(n)
	}))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "instantmeet_participants_active",
		Help: "Admitted participants across all meetings.",
	}, func() float64 {
		_, n, _ := c.MeetingStats()
		return float64(n)
	}))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "instantmeet_waiting_participants",
		Help: "Participants currently in waiting rooms.",
	}, func() float64 {
		_, _, n := c.MeetingStats()
		return float64(n)
	}))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "instantmeet_websocket_connections",
		Help: "Open meeting WebSocket connections.",
	}, func() float64 {
		return float64(c.WebSocketConnections())
	}))

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
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
		path := routePath(r)
		statusLabel := strconv.Itoa(status)
		m.requests.WithLabelValues(r.Method, path, statusLabel).Inc()
		m.duration.WithLabelValues(r.Method, path, statusLabel).Observe(duration.Seconds())
	})
}

// routePath prefers chi's matched pattern (low cardinality). Falls back to a
// coarse bucket when no pattern is available.
func routePath(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}
	switch {
	case r.URL.Path == "/ws":
		return "/ws"
	default:
		return "unmatched"
	}
}
