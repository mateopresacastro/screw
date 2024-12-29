package main

import (
	"log/slog"
	"os"
)

func main() {
	env := getEnv("ENV", "dev")
	if err := startServer(env); err != nil {
		slog.Error("Application failed", "err", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
