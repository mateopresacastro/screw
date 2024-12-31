package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"tagg/auth"
	"tagg/session"
	"tagg/store"
	"tagg/ws"

	_ "github.com/joho/godotenv/autoload"
)

const (
	port = 3000
	dir  = "/frontend/out"
)

func startServer(env string) error {
	port := fmt.Sprintf(":%d", port)
	mux := http.NewServeMux()
	if env == "prod" {
		fileServer := http.FileServer(http.Dir(dir))
		mux.Handle("/", fileServer)
		slog.Info("Registered static file server", "dir", dir)
	}
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
	mux.HandleFunc("/ws", ws.Ws)
	mux.HandleFunc("GET /login/google", google.HandleLogin)
	mux.HandleFunc("GET /login/google/callback", google.HandleCallBack)
	mux.HandleFunc("GET /login/session", google.HandleCurrentSession)
	mux.HandleFunc("POST /logout", google.HandleLogout)
	slog.Info("Server is listening", "port", port, "env", env)
	return http.ListenAndServe(port, mux)
}
