package main

import (
	"context"
	"os"
	"screw/server"
)

func main() {
	cfg := server.ServerCfg{
		Addr:         os.Getenv("ADDR"),
		ClientId:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Env:          os.Getenv("ENV"),
		DBPath:       "dev.db",
	}
	s := server.New(cfg)
	ctx := context.Background()
	s.Start(ctx)
}
