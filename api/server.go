package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"tagg/auth"
	"tagg/he"
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
		log.Panicln("something went wrong creating the store:", err)
	}

	sessionManager := session.NewManager(store, 30, 15, env == "prod")
	google := auth.NewGoogle(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		"http://localhost:3000/login/google/callback",
		store,
		sessionManager,
	)
	ws := ws.New(store)

	mux.Handle("/", http.FileServer(http.Dir(dir)))
	mux.Handle("/ws", he.AppHandler(ws.Handle))
	mux.Handle("GET /login/google", he.AppHandler(google.HandleLogin))
	mux.Handle("GET /login/google/callback", he.AppHandler(google.HandleCallBack))
	mux.Handle("GET /login/session", he.AppHandler(sessionManager.HandleCurrentSession))
	mux.Handle("POST /logout", he.AppHandler(sessionManager.HandleLogout))

	CORSAllowed := map[string]bool{
		"http://localhost:3001": true,
	}

	protectedRoutes := map[string]bool{
		"/login/session": true,
		"/logout":        true,
		"/ws":            true,
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
