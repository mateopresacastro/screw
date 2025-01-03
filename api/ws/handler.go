package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"tagg/ffmpeg"
	"tagg/he"
	"tagg/store"
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

type Metadata struct {
	FileSize int64   `json:"fileSize"`
	FileName string  `json:"fileName"`
	MimeType string  `json:"mimeType"`
	BPM      float32 `json:"bpm"`
}

type progressMessage struct {
	Type     string  `json:"type"`
	Progress float64 `json:"progress"`
}

func writeMessage(messageType int, data []byte, conn *websocket.Conn, writeMu *sync.Mutex) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return conn.WriteMessage(messageType, data)
}

type WS struct {
	store store.Store
}

func New(store store.Store) *WS {
	return &WS{store: store}
}

func (ws *WS) Handle(w http.ResponseWriter, r *http.Request) *he.AppError {
	slog.Info("New websocket connection - trying to upgrade")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return he.InternalError(err, "Failed to upgrade websocket connection")
	}
	defer conn.Close()
	slog.Info("Upgraded")

	err = conn.NetConn().SetDeadline(time.Now().Add(2 * time.Minute))
	if err != nil {
		return he.InternalError(err, "Error setting connection deadline")
	}

	messageType, message, err := conn.ReadMessage()
	if err != nil {
		return he.InternalError(err, "Error reading initial message")
	}

	var meta Metadata
	if messageType != websocket.TextMessage {
		return he.BadRequestError(errors.New("invalid message type"), "First message must be metadata")
	}

	if err := json.Unmarshal(message, &meta); err != nil {
		return he.BadRequestError(err, "Error parsing metadata")
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var writeMu sync.Mutex
	opts := ffmpeg.Options{
		BPM:           meta.BPM,
		BarsInterval:  2,
		DropOffset:    7,
		WatermarkGain: 0.5,
	}

	ffmpeg, err := ffmpeg.New(ctx, opts)
	if err != nil {
		return he.InternalError(err, "Failed to initialize ffmpeg")

	}
	defer func() {
		ffmpeg.Close()
		slog.Info("Websocket connection ended")
	}()

	go readWebSocketAndPipeToFFMPEG(ctx, ffmpeg, conn, &writeMu, meta.FileSize, meta.FileName)
	go readFFMPEGAndWriteToSocket(ctx, ffmpeg, conn, &writeMu)

	slog.Info("Listening to websocket. Waiting for processing completion or errors.")
	select {
	case err := <-ffmpeg.ErrChan:
		return he.InternalError(err, "Stream processing error")
	case <-ffmpeg.Done:
		slog.Info("Processing finished gracefully")
		return nil
	case <-ctx.Done():
		slog.Info("The context was cancelled")
		return nil
	}
}

func readFFMPEGAndWriteToSocket(
	ctx context.Context,
	ffmpeg *ffmpeg.FFMPEG,
	conn *websocket.Conn,
	writeMu *sync.Mutex,
) {
	buffer := make([]byte, 32*1024)
	processOutput := func() error {
		n, err := ffmpeg.Stdout.Read(buffer)
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
					ffmpeg.Done <- true
					return
				}
				ffmpeg.ErrChan <- err
				return
			}
		}
	}
}

func readWebSocketAndPipeToFFMPEG(
	ctx context.Context,
	ffmpeg *ffmpeg.FFMPEG,
	conn *websocket.Conn,
	writeMu *sync.Mutex,
	fileSize int64,
	fileName string,
) {
	var (
		receivedBytes int64
		lastProgress  float64
	)

	logTicker := time.NewTicker(2 * time.Second)
	defer logTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-logTicker.C:
			slog.Info("Processing",
				"name", fileName,
				"bytes", receivedBytes,
				"fileSize", fileSize,
				"progress", math.Round(lastProgress))
			continue
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
					ffmpeg.Done <- true
					return
				}
				ffmpeg.ErrChan <- fmt.Errorf("websocket read error: %w", err)
				return
			}

			if messageType != websocket.BinaryMessage {
				ffmpeg.ErrChan <- fmt.Errorf("unexpected message type: %v", messageType)
				return
			}

			if _, err := ffmpeg.Write(message); err != nil {
				slog.Error("Error while writing to ffmpeg stdin", "err", err)
				ffmpeg.ErrChan <- fmt.Errorf("error while writing to ffmpeg stdin: %w", err)
				return
			}

			receivedBytes += int64(len(message))
			progress := float64(receivedBytes) / float64(fileSize) * 100
			progressMsg := progressMessage{
				Type:     "progress",
				Progress: progress,
			}

			progressJSON, err := json.Marshal(progressMsg)
			if err != nil {
				ffmpeg.ErrChan <- fmt.Errorf("error parsing progress message: %w", err)
				return
			}
			if err := writeMessage(websocket.TextMessage, progressJSON, conn, writeMu); err != nil {
				ffmpeg.ErrChan <- fmt.Errorf("Failed to send progress: %w", err)
				return
			}
			lastProgress = progress
			continue
		}
	}
}
