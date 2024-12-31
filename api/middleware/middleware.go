package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"tagg/session"
	"time"

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
			slog.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"ip", r.RemoteAddr,
				"duration", time.Since(start),
			)
		})
	}
}

func CORS(allowedOrigins map[string]struct{}) Middleware {
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

func Protect(protectedRoutes map[string]struct{}, sm *session.Manager) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, protected := protectedRoutes[r.URL.Path]; !protected {
				next.ServeHTTP(w, r)
				return
			}

			result, err := sm.GetCurrentSession(r)
			if err != nil {
				slog.Error("error getting session", "error", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if result == nil || result.User == nil {
				slog.Error("no active session")
				http.Error(w, "No active session", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RateLimit(rps float64, burst int) Middleware {
	limiters := &sync.Map{}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" {
				// Handle IPv6 [::1]:port format
				ip = r.RemoteAddr
				if strings.Contains(ip, "[") {
					ip = strings.Split(strings.Split(ip, "]")[0], "[")[1]
				} else {
					ip = strings.Split(ip, ":")[0]
				}
			}

			limiter, _ := limiters.LoadOrStore(ip, rate.NewLimiter(rate.Limit(rps), burst))
			l := limiter.(*rate.Limiter)
			if !l.Allow() {
				slog.Error("too many requests")
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			slog.Info("rate limiter state",
				"ip", ip,
				"tokens", l.Tokens(),
				"limit", l.Limit(),
				"burst", l.Burst(),
			)
			next.ServeHTTP(w, r)
		})
	}
}
