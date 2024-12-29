package main

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func writeMessage(messageType int, data []byte, conn *websocket.Conn, writeMu *sync.Mutex) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return conn.WriteMessage(messageType, data)
}

func ws(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection", "error", err)
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer conn.Close()
	defer cancel()

	slog.Info("New websocket connection")

	errChan := make(chan error, 3)
	var writeMu sync.Mutex

	ffmpegStdin, ffmpegStdout, ffmpegStderr, err := ffmpeg(ctx)
	if err != nil {
		return
	}

	go handleFFMPEGInput(ctx, ffmpegStdin, errChan, conn, &writeMu, cancel)
	go handleFFMPEGOutput(ctx, ffmpegStdout, errChan, conn, &writeMu, cancel)
	go handleFFMPEGErr(ffmpegStderr, errChan)

	slog.Info("Waiting for completion or errors")
	select {
	case err := <-errChan:
		slog.Error("Stream processing error", "error", err)
	case <-ctx.Done():
		slog.Info("Stream processing completed successfully")
	}
	slog.Info("Websocket connection ended")
}
