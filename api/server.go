package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"tagg/auth"
	mw "tagg/middleware"
	"tagg/session"
	"tagg/store"
	"tagg/ws"

	_ "github.com/joho/godotenv/autoload"
)

const (
	port = 3000
	dir  = "/frontend/out"
)

var portStr = fmt.Sprintf(":%d", port)

func startServer(env string) error {
	mux := http.NewServeMux()
	store, err := store.NewFromEnv(env)
	if err != nil {
		log.Panicln(err)
		panic("Something went wrong creating the store")
	}

	sessionManager := session.NewManager(store, 30, 15, env == "prod")
	google := auth.NewGoogle(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		"http://localhost:3000/login/google/callback",
		store,
		sessionManager,
	)

	mux.Handle("/", http.FileServer(http.Dir(dir)))
	mux.HandleFunc("/ws", ws.Handler)
	mux.HandleFunc("GET /login/google", google.HandleLogin)
	mux.HandleFunc("GET /login/google/callback", google.HandleCallBack)
	mux.HandleFunc("GET /login/session", sessionManager.HandleCurrentSession)
	mux.HandleFunc("POST /logout", sessionManager.HandleLogout)

	CORSAllowed := map[string]struct{}{
		"http://localhost:3001": {},
	}

	protectedRoutes := map[string]struct{}{
		"/login/session": {},
		"/logout":        {},
		"/ws":            {},
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
