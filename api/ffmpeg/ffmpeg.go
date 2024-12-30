package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
)

type FFMPEG struct {
	Stdin   io.WriteCloser
	Stdout  io.ReadCloser
	Stderr  io.ReadCloser
	Ctx     context.Context
	ErrChan chan error
	Done    chan struct{}
}

func New(ctx context.Context, errChan chan error, done chan struct{}) (*FFMPEG, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
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
	f := &FFMPEG{
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Ctx:     ctx,
		ErrChan: errChan,
		Done:    done,
	}
	go f.monitor()
	return f, nil
}

func (f *FFMPEG) monitor() {
	buf := make([]byte, 1024)
	for {
		select {
		case <-f.Ctx.Done():
			return
		default:
			n, err := f.Stderr.Read(buf)
			if n > 0 {
				f.ErrChan <- fmt.Errorf("ffmpeg error: %s", string(buf[:n]))
				return
			}
			if err != nil {
				if err != io.EOF {
					f.ErrChan <- fmt.Errorf("stderr read error: %w", err)
				}
				return
			}
		}
	}
}

func (f *FFMPEG) Close() {
	f.Stdin.Close()
	f.Stdout.Close()
	f.Stderr.Close()
	close(f.ErrChan)
	close(f.Done)
	slog.Info("ffmpeg clean up done! All good!")
}

func (f *FFMPEG) Write(p []byte) (n int, err error) {
	n, err = f.Stdin.Write(p)
	if err != nil {
		f.ErrChan <- err
	}
	return n, err
}

func (f *FFMPEG) Read(p []byte) (n int, err error) {
	n, err = f.Stdout.Read(p)
	if err != nil {
		if err == io.EOF {
			f.Done <- struct{}{}
		}
		f.ErrChan <- err
	}
	return n, err
}
