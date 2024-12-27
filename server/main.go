package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Application failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	env := getEnv("ENV", "dev")
	if env == "prod" {
		go startFileServer()
	}

	http.HandleFunc("GET /api", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	slog.Info("API listening on port 3000")
	return http.ListenAndServe(":3000", nil)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func startFileServer() error {
	dir := "/client/dist"
	http.Handle("GET /", http.FileServer(http.Dir(dir)))
	slog.Info("Starting file server", "env", "prod", "port", "3000")
	return http.ListenAndServe(":3000", nil)
}
