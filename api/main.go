package main

import (
	"log/slog"
	"os"
	"screw/server"
)

func main() {
	cfg := server.ServerCfg{
		Host:         os.Getenv("HOST"),
		ClientId:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		Env:          os.Getenv("ENV"),
		DBPath:       "dev.db",
	}
	s := server.New(cfg)
	err := s.Start()
	if err != nil {
		slog.Error("Application failed", "err", err)
		os.Exit(1)
	}
}
