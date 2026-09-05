package media

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gotask/gotask/internal/jobs"
	xdraw "golang.org/x/image/draw"
)

// Register wires the PixelForge handlers into the existing GoTask registry.
// Media jobs then flow through the same queue, worker pool, retry and
// persistence path as every other job type.
func (m *Manager) Register(registry *jobs.Registry) {
	registry.Register(JobImageMetadata, jobs.HandlerFunc(handleImageMetadata))
	registry.Register(JobImageThumbnail, jobs.HandlerFunc(imageResizeHandler(320, 80)))
	registry.Register(JobImageResize, jobs.HandlerFunc(imageResizeHandler(1280, 85)))
	registry.Register(JobImageCompress, jobs.HandlerFunc(imageResizeHandler(0, 60)))
	registry.Register(JobImageOptimize, jobs.HandlerFunc(imageResizeHandler(1920, 70)))

	registry.Register(JobVideoMetadata, jobs.HandlerFunc(m.handleVideoMetadata))
	registry.Register(JobVideoThumbnail, jobs.HandlerFunc(m.videoFFmpegHandler(func(in, out string) []string {
		return []string{"-y", "-ss", "00:00:01", "-i", in, "-frames:v", "1", "-q:v", "3", out}
	})))
	registry.Register(JobVideoAudio, jobs.HandlerFunc(m.videoFFmpegHandler(func(in, out string) []string {
		return []string{"-y", "-i", in, "-vn", "-c:a", "aac", "-b:a", "192k", out}
	})))
	for jobType, height := range map[string]string{
		JobVideo1080p: "1080",
		JobVideo720p:  "720",
		JobVideo480p:  "480",
	} {
		h := height
		registry.Register(jobType, jobs.HandlerFunc(m.videoFFmpegHandler(func(in, out string) []string {
			return []string{"-y", "-i", in, "-vf", "scale=-2:" + h, "-c:v", "libx264",
				"-preset", "veryfast", "-crf", "23", "-c:a", "aac", out}
		})))
	}
}

type imageMetadata struct {
	Filename  string `json:"filename"`
	Format    string `json:"format"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	SizeBytes int64  `json:"size_bytes"`
	Engine    string `json:"engine"`
}

func handleImageMetadata(ctx context.Context, job *jobs.Job) error {
	p, err := decodePayload(job.Payload)
	if err != nil {
		return jobs.NewPermanentError(err)
	}

	f, err := os.Open(p.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source image: %w", err)
	}
	defer f.Close()

	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return jobs.NewPermanentError(fmt.Errorf("failed to decode image: %w", err))
	}

	stat, err := os.Stat(p.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source image: %w", err)
	}

	return writeJSONFile(p.OutputPath, imageMetadata{
		Filename:  p.Filename,
		Format:    format,
		Width:     cfg.Width,
		Height:    cfg.Height,
		SizeBytes: stat.Size(),
		Engine:    "go-image",
	})
}

// imageResizeHandler decodes, scales to fit maxDim (0 keeps original size) and
// re-encodes as JPEG at the given quality. This is genuine processing — no
// external binary required.
func imageResizeHandler(maxDim int, quality int) func(context.Context, *jobs.Job) error {
	return func(ctx context.Context, job *jobs.Job) error {
		p, err := decodePayload(job.Payload)
		if err != nil {
			return jobs.NewPermanentError(err)
		}

		src, err := os.Open(p.SourcePath)
		if err != nil {
			return fmt.Errorf("failed to open source image: %w", err)
		}
		defer src.Close()

		img, _, err := image.Decode(src)
		if err != nil {
			return jobs.NewPermanentError(fmt.Errorf("failed to decode image: %w", err))
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		bounds := img.Bounds()
		width, height := bounds.Dx(), bounds.Dy()
		if maxDim > 0 && (width > maxDim || height > maxDim) {
			if width > height {
				height = height * maxDim / width
				width = maxDim
			} else {
				width = width * maxDim / height
				height = maxDim
			}
			scaled := image.NewRGBA(image.Rect(0, 0, width, height))
			xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), img, bounds, xdraw.Over, nil)
			img = scaled
		}

		out, err := os.Create(p.OutputPath)
		if err != nil {
			return fmt.Errorf("failed to create output image: %w", err)
		}
		defer out.Close()

		if err := jpeg.Encode(out, img, &jpeg.Options{Quality: quality}); err != nil {
			return fmt.Errorf("failed to encode output image: %w", err)
		}
		return nil
	}
}

type videoMetadata struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	Container string `json:"container"`
	Engine    string `json:"engine"`
	Probe     string `json:"probe,omitempty"`
	Note      string `json:"note,omitempty"`
}

func (m *Manager) handleVideoMetadata(ctx context.Context, job *jobs.Job) error {
	p, err := decodePayload(job.Payload)
	if err != nil {
		return jobs.NewPermanentError(err)
	}

	stat, err := os.Stat(p.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source video: %w", err)
	}

	meta := videoMetadata{
		Filename:  p.Filename,
		SizeBytes: stat.Size(),
		Container: strings.TrimPrefix(strings.ToLower(pathExt(p.SourcePath)), "."),
		Engine:    "basic",
		Note:      "install ffmpeg for full stream metadata",
	}

	if m.ffprobePath != "" {
		cmd := exec.CommandContext(ctx, m.ffprobePath, "-v", "quiet", "-print_format", "json",
			"-show_format", "-show_streams", p.SourcePath)
		out, probeErr := cmd.Output()
		if probeErr != nil {
			return fmt.Errorf("ffprobe failed: %w", probeErr)
		}
		meta.Engine = "ffprobe"
		meta.Note = ""
		meta.Probe = string(out)
	}

	return writeJSONFile(p.OutputPath, meta)
}

// videoFFmpegHandler runs a real ffmpeg command when ffmpeg is installed.
// Without it there is no honest way to transcode in pure Go, so the job still
// runs through the full queue/worker path and records that its output was
// simulated rather than pretending a real transcode happened.
func (m *Manager) videoFFmpegHandler(args func(in, out string) []string) func(context.Context, *jobs.Job) error {
	return func(ctx context.Context, job *jobs.Job) error {
		p, err := decodePayload(job.Payload)
		if err != nil {
			return jobs.NewPermanentError(err)
		}

		if m.ffmpegPath == "" {
			return simulateVideoWork(ctx, p)
		}

		cmd := exec.CommandContext(ctx, m.ffmpegPath, args(p.SourcePath, p.OutputPath)...)
		output, runErr := cmd.CombinedOutput()
		if runErr != nil {
			detail := lastLines(string(output), 3)
			if strings.Contains(detail, "does not contain any stream") ||
				strings.Contains(detail, "Output file does not contain any stream") {
				return jobs.NewPermanentError(fmt.Errorf("source has no matching stream: %s", detail))
			}
			return fmt.Errorf("ffmpeg failed: %v: %s", runErr, detail)
		}
		return nil
	}
}

func simulateVideoWork(ctx context.Context, p *Payload) error {
	delay := time.Duration(400+rand.Intn(600)) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return ctx.Err()
	}

	return writeJSONFile(p.OutputPath+".simulated.json", map[string]string{
		"status":  "simulated",
		"reason":  "ffmpeg is not installed on this host",
		"job":     p.Variant,
		"source":  p.Filename,
		"install": "install ffmpeg and restart the server for real transcoding",
	})
}

func pathExt(p string) string {
	if i := strings.LastIndex(p, "."); i >= 0 {
		return p[i:]
	}
	return ""
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.TrimSpace(strings.Join(lines, " "))
}
