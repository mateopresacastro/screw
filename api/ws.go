package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

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
	// you can't write concurrently to a websocket se we need to use a mutex
	writeMu.Lock()
	defer writeMu.Unlock()
	return conn.WriteMessage(messageType, data)
}

func ws(w http.ResponseWriter, r *http.Request) {
	slog.Info("New websocket connection! Trying to upgrade...")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection", "error", err)
		return
	}
	defer conn.Close()
	slog.Info("Upgraded")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	err = conn.NetConn().SetDeadline(time.Now().Add(time.Minute))
	if err != nil {
		slog.Error("Error setting connection deadline", "error", err)
		return
	}

	var writeMu sync.Mutex
	errChan := make(chan error, 3)
	done := make(chan struct{})

	ffmpeg, err := newFFMPEG(ctx, errChan, done)
	if err != nil {
		return
	}
	defer ffmpeg.close()

	go readWebSocketAnPipeToFFMPEG(ctx, ffmpeg, conn, &writeMu)
	go readFFMPEGAndWriteToSocket(ctx, ffmpeg, conn, &writeMu)

	slog.Info("Listening to websocket. Waiting for processing completion or errors.")
	select {
	case err := <-errChan:
		slog.Error("Stream processing error", "error", err)
		cancel()
	case <-done:
		slog.Info("Processing finished gracefully")
	case <-ctx.Done():
		slog.Info("The context was cancelled")
	}
	slog.Info("Websocket connection ended")
}

func readFFMPEGAndWriteToSocket(
	ctx context.Context,
	ffmpeg *ffmpeg,
	conn *websocket.Conn,
	writeMu *sync.Mutex,
) {
	buffer := make([]byte, 32*1024)
	processOutput := func() error {
		n, err := ffmpeg.stdout.Read(buffer)
		if err != nil {
			return err
		}
		if err := writeMessage(websocket.BinaryMessage, buffer[:n], conn, writeMu); err != nil {
			return err
		}
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := processOutput(); err != nil {
				if err == io.EOF {
					ffmpeg.done <- struct{}{}
					return
				}
				ffmpeg.errChan <- err
				return
			}
		}
	}
}

func readWebSocketAnPipeToFFMPEG(
	ctx context.Context,
	ffmpeg *ffmpeg,
	conn *websocket.Conn,
	writeMu *sync.Mutex,
) {
	var receivedBytes int64
	var fileSize int64 = 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(
					err,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNoStatusReceived,
				) {
					slog.Info("WebSocket closed normally by client")
					ffmpeg.done <- struct{}{}
					return
				}
				ffmpeg.errChan <- fmt.Errorf("websocket read error: %w", err)
				return
			}
			if messageType == websocket.TextMessage {
				var metadata Metadata
				if err := json.Unmarshal(message, &metadata); err != nil {
					ffmpeg.errChan <- fmt.Errorf("error parsing metadata from websocket: %w", err)
				}
				fileSize = metadata.FileSize
				slog.Info("Received file metadata",
					"size", metadata.FileSize,
					"name", metadata.FileName)
				continue
			}
			if messageType == websocket.BinaryMessage {
				if _, err := ffmpeg.write(message); err != nil {
					slog.Error("error while writing to ffmpeg stdin", "err", err)
					return
				}
				receivedBytes += int64(len(message))
				progress := float64(receivedBytes) / float64(fileSize) * 100
				progressMsg := ProgressMessage{
					Type:     "progress",
					Progress: progress,
				}
				progressJSON, err := json.Marshal(progressMsg)
				if err != nil {
					ffmpeg.errChan <- fmt.Errorf("error parsing progress message: %w", err)
					return
				}
				if err := writeMessage(websocket.TextMessage, progressJSON, conn, writeMu); err != nil {
					ffmpeg.errChan <- fmt.Errorf("Failed to send progress: %w", err)
					return
				}
				slog.Info("Processing...",
					"received", receivedBytes,
					"total", fileSize,
					"progress", progress)
			}
		}
	}
}
