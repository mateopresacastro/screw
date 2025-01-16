package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"tagg/auth"
	"tagg/hr"
	mw "tagg/middleware"
	"tagg/session"
	"tagg/store"
	"tagg/ws"

	_ "github.com/joho/godotenv/autoload"
)

const (
	port = 3000
)

var portStr = fmt.Sprintf(":%d", port)

func startServer(env string) error {
	mux := http.NewServeMux()
	store, err := store.NewFromEnv(env)
	if err != nil {
		log.Panicln("something went wrong creating the store:", err)
	}

	sessionManager := session.NewManager(store, 30, 15, env == "prod")
	ws := ws.New(store)
	google := auth.NewGoogle(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		"http://localhost/api/login/google/callback",
		store,
		sessionManager,
	)
	mux.Handle("/api/ws", hr.W(ws.Handle))
	mux.Handle("GET /api/login/google", hr.W(google.HandleLogin))
	mux.Handle("GET /api/login/google/callback", hr.W(google.HandleCallBack))
	mux.Handle("GET /api/login/session", hr.W(sessionManager.HandleCurrentSession))
	mux.Handle("POST /api/logout", hr.W(sessionManager.HandleLogout))
	mux.Handle("GET /api/healthcheck", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	CORSAllowed := map[string]bool{
		"http://localhost:3001": true,
		"http://localhost":      true,
	}

	protectedRoutes := map[string]bool{
		"/api/login/session": true,
		"/api/logout":        true,
	}

	server := mw.Chain(
		mux,
		mw.RateLimit(15, 50), // add 15 requests per second to bucket, 50 in burst for chunk request
		mw.Logger(),
		mw.CORS(CORSAllowed),
		mw.Protect(protectedRoutes, sessionManager),
	)

	slog.Info("Server is listening", "port", port, "env", env)
	return http.ListenAndServe(portStr, server)
}
