package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

func ffmpeg(ctx context.Context) (io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", "pipe:0",
		"-af", "rubberband=pitch=0.80",
		"-c:a", "aac",
		"-f", "adts",
		"-movflags", "empty_moov",
		"pipe:1",
	)

	ffmpegStdin, err := cmd.StdinPipe()
	if err != nil {
		slog.Error("Failed to create stdin pipe", "error", err)
		return nil, nil, nil, err
	}

	ffmpegStdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Failed to create stdout pipe", "error", err)
		return nil, nil, nil, err
	}

	ffmpegStderr, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("Failed to create stderr pipe", "error", err)
		return nil, nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start FFmpeg", "error", err)
		return nil, nil, nil, err
	}

	return ffmpegStdin, ffmpegStdout, ffmpegStderr, err
}

func handleFFMPEGErr(
	ffmpegStderr io.ReadCloser,
	errChan chan error,
) {
	scanner := bufio.NewScanner(ffmpegStderr)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "Error") ||
			strings.Contains(scanner.Text(), "Invalid") {
			errChan <- fmt.Errorf("stderr scanner error: %s", scanner.Text())
		} else {
			slog.Debug("FFmpeg output", "message", scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		errChan <- fmt.Errorf("stderr scanner error: %w", err)
	}
}

func handleFFMPEGOutput(
	ctx context.Context,
	ffmpegStdout io.ReadCloser,
	errChan chan error,
	conn *websocket.Conn,
	writeMu *sync.Mutex,
	cancel context.CancelFunc,
) {
	slog.Info("Starting FFmpeg output handler")
	buffer := make([]byte, 32*1024)
	defer func() {
		ffmpegStdout.Close()
	}()

	processOutput := func() error {
		n, err := ffmpegStdout.Read(buffer)
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
					cancel()
					return
				}
				errChan <- err
				return
			}
		}
	}
}

func handleFFMPEGInput(
	ctx context.Context,
	ffmpegStdin io.WriteCloser,
	errChan chan error,
	conn *websocket.Conn,
	writeMu *sync.Mutex,
	cancel context.CancelFunc,
) {
	slog.Info("Starting WebSocket input handler")
	var receivedBytes int64
	var fileSize int64 = -1

	defer func() {
		slog.Info("Closing stdin pipe")
		ffmpegStdin.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					slog.Info("WebSocket closed normally by client")
					cancel()
					continue
				}
				errChan <- fmt.Errorf("websocket read error: %w", err)
				return
			}

			if messageType == websocket.TextMessage {
				var metadata Metadata
				if err := json.Unmarshal(message, &metadata); err != nil {
					errChan <- fmt.Errorf("error parsing metadata from websocket: %w", err)
				}
				fileSize = metadata.FileSize
				slog.Info("Received file metadata",
					"size", metadata.FileSize,
					"name", metadata.FileName)
				continue
			}

			if messageType == websocket.BinaryMessage {
				if _, err := ffmpegStdin.Write(message); err != nil {
					errChan <- fmt.Errorf("error while writing to ffmpeg stdin: %w", err)
					return
				}

				receivedBytes += int64(len(message))
				progress := float64(receivedBytes) / float64(fileSize) * 100
				progressMsg := ProgressMessage{
					Type:         "progress",
					Progress:     progress,
					ReceivedSize: receivedBytes,
					TotalSize:    fileSize,
				}

				progressJSON, err := json.Marshal(progressMsg)
				if err != nil {
					errChan <- fmt.Errorf("error parsing progress message: %w", err)
					return
				}
				if err := writeMessage(websocket.TextMessage, progressJSON, conn, writeMu); err != nil {
					errChan <- fmt.Errorf("Failed to send progress: %w", err)
					return
				}

				slog.Info("Processing progress",
					"receivedBytes", receivedBytes,
					"totalBytes", fileSize,
					"progress", progress)
			}
		}
	}
}
