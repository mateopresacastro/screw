package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
)

type FFMPEG struct {
	Stdin   io.WriteCloser
	Stdout  io.ReadCloser
	Stderr  io.ReadCloser
	Ctx     context.Context
	ErrChan chan error
	Done    chan struct{}
}

type Options struct {
	BPM           float32
	BarsInterval  int
	DropOffset    float64
	WatermarkGain float64
}

func New(ctx context.Context, errChan chan error, done chan struct{}, tagPath string, opts Options) (*FFMPEG, error) {
	tagDuration, err := getTagDuration(tagPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag duration: %w", err)
	}

	// Calculate musical timing
	beatsPerSecond := float64(opts.BPM) / 60
	beatDuration := 1 / beatsPerSecond
	barDuration := 4 * beatDuration // Assuming 4/4 time signature

	// Calculate precise timing for the drop
	dropOffsetSeconds := opts.DropOffset * beatDuration

	// Total cycle duration in seconds
	totalCycleDuration := float64(opts.BarsInterval) * barDuration

	filterComplex := fmt.Sprintf(
		"[1:a]volume=%.3f[watermark];"+
			"[watermark]asetpts=PTS-STARTPTS,"+
			"adelay=%d|%d[delayed];"+
			"[delayed]aformat=sample_fmts=fltp:sample_rates=44100,"+
			"aselect=expr='between(mod(t-%.3f,%f),0,%.3f)'[periodic];"+
			"[0:a][periodic]amix=inputs=2:duration=first:weights=1 %.3f[out]",
		opts.WatermarkGain,
		int(dropOffsetSeconds*1000), int(dropOffsetSeconds*1000), // Delay in milliseconds
		dropOffsetSeconds, totalCycleDuration, tagDuration,
		opts.WatermarkGain,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", "pipe:0",
		"-stream_loop", "-1",
		"-i", tagPath,
		"-filter_complex", filterComplex,
		"-map", "[out]",
		"-c:a", "aac",
		"-b:a", "192k",
		"-ar", "44100",
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
	slog.Info("ffmpeg clean up done! All good!")
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
			f.Done <- struct{}{}
		}
		f.ErrChan <- err
	}
	return n, err
}

func getTagDuration(filepath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filepath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
}
