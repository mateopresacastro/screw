package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

type FFMPEG struct {
	Stdin   io.WriteCloser
	Stdout  io.ReadCloser
	Stderr  io.ReadCloser
	Ctx     context.Context
	ErrChan chan error
	Done    chan bool
}

func New(ctx context.Context) (*FFMPEG, error) {
	irPath := "api/audio/ir.wav"

	if envPath := os.Getenv("IR_PATH"); envPath != "" {
		irPath = envPath
	}

	absPath, err := filepath.Abs(irPath)
	if err != nil {
		slog.Error("Failed to get absolute path for IR", "error", err)
		return nil, err
	}

	slog.Info("Using IR file", "path", absPath)

	filterComplex := "[0:a][1:a]afir=dry=10:wet=10[reverbed];[reverbed]highpass=f=40,lowpass=f=2300[filtered];[filtered]asetrate=44100*0.9,aresample=44100,atempo=0.97[out]"

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", "pipe:0", // Main audio
		"-i", absPath, // IR file
		"-filter_complex", filterComplex,
		"-map", "[out]",
		"-c:a", "aac",
		"-b:a", "256k",
		"-f", "adts",
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

	errChan := make(chan error, 3)
	done := make(chan bool)

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
	slog.Info("Ffmpeg clean up done.")
}

func (f *FFMPEG) Write(p []byte) (int, error) {
	n, err := f.Stdin.Write(p)
	if err != nil {
		f.ErrChan <- err
	}
	return n, err
}

func (f *FFMPEG) Read(p []byte) (int, error) {
	n, err := f.Stdout.Read(p)
	if err != nil {
		if err == io.EOF {
			f.Done <- true
		}
		f.ErrChan <- err
	}
	return n, err
}
