package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
)

type ffmpeg struct {
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	ctx     context.Context
	errChan chan error
	done    chan struct{}
}

func newFFMPEG(ctx context.Context, errChan chan error, done chan struct{}) (*ffmpeg, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", "pipe:0",
		"-af", "rubberband=pitch=0.80",
		"-c:a", "aac",
		"-f", "adts",
		"-movflags", "empty_moov",
		"pipe:1",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		slog.Error("Failed to create stdin pipe", "error", err)
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Failed to create stdout pipe", "error", err)
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("Failed to create stderr pipe", "error", err)
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start FFmpeg", "error", err)
		return nil, err
	}
	newFFMPEG := &ffmpeg{
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		ctx:     ctx,
		errChan: errChan,
		done:    done,
	}
	go newFFMPEG.listenForErrors()
	return newFFMPEG, nil
}

func (f *ffmpeg) listenForErrors() {
	scanner := bufio.NewScanner(f.stderr)
	go func() {
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Error") ||
				strings.Contains(scanner.Text(), "Invalid") {
				f.errChan <- fmt.Errorf("stderr scanner error: %s", scanner.Text())
			} else {
				slog.Debug("ffmpeg output", "message", scanner.Text())
			}
		}
	}()

	if err := scanner.Err(); err != nil {
		f.errChan <- fmt.Errorf("stderr scanner error: %w", err)
	}
}

func (f *ffmpeg) close() {
	if f.stdin != nil {
		f.stdin.Close()
	}
	if f.stdout != nil {
		f.stdout.Close()
	}
	if f.stderr != nil {
		f.stderr.Close()
	}
	slog.Info("ffmpeg clean up done! All good!")
}

func (f *ffmpeg) write(p []byte) (n int, err error) {
	n, err = f.stdin.Write(p)
	if err != nil {
		f.errChan <- err
	}
	return n, err
}

func (f *ffmpeg) read(p []byte) (n int, err error) {
	n, err = f.stdout.Read(p)
	if err != nil {
		if err == io.EOF {
			f.done <- struct{}{}
		}
		f.errChan <- err
	}
	return n, err
}
