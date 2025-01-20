package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"screw/ffmpeg"
	"screw/herr"
	"screw/store"
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

type Metadata struct {
	FileSize int64  `json:"fileSize"`
	FileName string `json:"fileName"`
	MimeType string `json:"mimeType"`
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

func (ws *WS) Handle(w http.ResponseWriter, r *http.Request) *herr.Error {
	slog.Info("New websocket connection - trying to upgrade")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return herr.Internal(err, "Failed to upgrade websocket connection")
	}
	defer conn.Close()
	slog.Info("Upgraded")

	err = conn.NetConn().SetDeadline(time.Now().Add(5 * time.Minute))
	if err != nil {
		herr.WS(conn, err, "Connection deadline error")
		return nil
	}

	messageType, message, err := conn.ReadMessage()
	if err != nil {
		herr.WS(conn, err, "Error reading first message")
		return nil
	}

	if messageType != websocket.TextMessage {
		herr.WS(conn, err, "First message must be metadata")
		return nil
	}

	var meta Metadata
	if err := json.Unmarshal(message, &meta); err != nil {
		herr.WS(conn, err, "Error initializing ffmpeg")
		return nil
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var writeMu sync.Mutex

	ffmpeg, err := ffmpeg.New(ctx)
	if err != nil {
		herr.WS(conn, err, "Error initializing ffmpeg")
		return nil
	}

	defer func() {
		ffmpeg.Close()
		slog.Info("Websocket connection ended")
	}()
	readDone := make(chan struct{})
	writeDone := make(chan struct{})

	go readWebSocketAndPipeToFFMPEG(ctx, ffmpeg, conn, &writeMu, meta.FileSize, meta.FileName, readDone)
	go readFFMPEGAndWriteToSocket(ctx, ffmpeg, conn, &writeMu, writeDone)

	slog.Info("Listening to websocket. Waiting for processing completion or errors.")
	select {
	case err := <-ffmpeg.ErrChan:
		cancel()
		<-readDone
		<-writeDone
		herr.WS(conn, err, "Stream processing error")
		return nil
	case <-ffmpeg.Done:
		cancel()
		<-readDone
		<-writeDone
		herr.WSClose(conn, "Processing complete")
		return nil
	case <-ctx.Done():
		<-readDone
		<-writeDone
		slog.Info("The context was cancelled")
		return nil
	}
}

func readFFMPEGAndWriteToSocket(
	ctx context.Context,
	ffmpeg *ffmpeg.FFMPEG,
	conn *websocket.Conn,
	writeMu *sync.Mutex,
	done chan struct{},
) {
	defer close(done)
	buffer := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := ffmpeg.Stdout.Read(buffer)
			if err != nil {
				if err == io.EOF {
					ffmpeg.Done <- true
					return
				}
				ffmpeg.ErrChan <- err
				return
			}
			if err := writeMessage(websocket.BinaryMessage, buffer[:n], conn, writeMu); err != nil {
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
	done chan struct{},
) {
	defer close(done)
	var receivedBytes int64
	var lastProgress float64

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
