package middleware

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"screw/session"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/time/rate"
)

type Middleware func(http.Handler) http.Handler

func Chain(router *http.ServeMux, m ...Middleware) http.Handler {
	var handler http.Handler = router
	for i := len(m) - 1; i >= 0; i-- {
		handler = m[i](handler)
	}
	return handler
}

func Logger() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			slog.Info("Request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"ip", r.RemoteAddr,
				"duration", time.Since(start),
			)
		})
	}
}

func CORS(allowedOrigins map[string]bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			_, allowed := allowedOrigins[origin]
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func Protect(protectedRoutes map[string]bool, sm *session.Manager) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, protected := protectedRoutes[r.URL.Path]; !protected {
				next.ServeHTTP(w, r)
				return
			}

			result, err := sm.GetCurrentSession(r)
			if err != nil {
				slog.Error("Error getting session", "error", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if result == nil || result.User == nil {
				slog.Error("No active session")
				http.Error(w, "No active session", http.StatusUnauthorized)
				return
			}
			// add user data to context
			ctx := context.WithValue(r.Context(), session.SessionContextKey, result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RateLimit(rps float64, burst int) Middleware {
	type limiterEntry struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	limiters := &sync.Map{}

	cleanup := time.NewTicker(5 * time.Minute)
	go func() {
		for range cleanup.C {
			now := time.Now()
			slog.Info("Cheking rate limiter stored ips")
			limiters.Range(func(key, value any) bool {
				entry := value.(*limiterEntry)
				if now.Sub(entry.lastSeen) > time.Hour {
					slog.Info("Found 1 hour old IP on rate limiter - deleting")
					limiters.Delete(key)
				}
				return true
			})
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" {
				ip = r.RemoteAddr
				if strings.Contains(ip, "[") {
					ip = strings.Split(strings.Split(ip, "]")[0], "[")[1]
				} else {
					ip = strings.Split(ip, ":")[0]
				}
			}

			var entry *limiterEntry
			value, loaded := limiters.Load(ip)
			if !loaded {
				entry = &limiterEntry{
					limiter:  rate.NewLimiter(rate.Limit(rps), burst),
					lastSeen: time.Now(),
				}
				limiters.Store(ip, entry)
			} else {
				entry = value.(*limiterEntry)
				entry.lastSeen = time.Now()
			}

			if !entry.limiter.Allow() {
				slog.Error("Too many requests", "ip", ip)
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			slog.Info("Rate limiter state",
				"ip", ip,
				"tokens", entry.limiter.Tokens(),
				"limit", entry.limiter.Limit(),
				"burst", entry.limiter.Burst(),
			)

			next.ServeHTTP(w, r)
		})
	}
}

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	wsConnectionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ws_connections_total",
			Help: "Total number of WebSocket connections",
		},
	)
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("hijacking not supported")
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Metrics() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			duration := time.Since(start).Seconds()
			httpRequestDuration.WithLabelValues(r.URL.Path, r.Method).Observe(duration)
			httpRequestsTotal.WithLabelValues(r.URL.Path, r.Method, fmt.Sprintf("%d", rw.statusCode)).Inc()
		})
	}
}
