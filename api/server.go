package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"tagg/ws"
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
	mux.HandleFunc("/ws", ws.Ws)
	slog.Info("Server is listening", "port", port, "env", env)
	return http.ListenAndServe(port, mux)
}
