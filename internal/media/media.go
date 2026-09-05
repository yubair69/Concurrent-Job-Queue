// Package media turns a single uploaded file into a set of GoTask jobs and
// provides the handlers that execute them. It owns no queue or worker logic of
// its own — every job it creates runs through the existing GoTask engine.
package media

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
)

type Manager struct {
	UploadDir      string
	OutputDir      string
	MaxUploadBytes int64

	ffmpegPath  string
	ffprobePath string
}

func NewManager(uploadDir, outputDir string, maxUploadMB int) *Manager {
	m := &Manager{
		UploadDir:      uploadDir,
		OutputDir:      outputDir,
		MaxUploadBytes: int64(maxUploadMB) * 1024 * 1024,
	}
	m.ffmpegPath, _ = exec.LookPath("ffmpeg")
	m.ffprobePath, _ = exec.LookPath("ffprobe")
	return m
}

// HasFFmpeg reports whether real video processing is available on this host.
func (m *Manager) HasFFmpeg() bool { return m.ffmpegPath != "" }

var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".webp": true,
}

var videoExtensions = map[string]bool{
	".mp4": true, ".mov": true, ".mkv": true, ".webm": true, ".avi": true, ".m4v": true,
}

// ClassifyMedia maps a filename to "image", "video", or "" if unsupported.
func ClassifyMedia(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch {
	case imageExtensions[ext]:
		return MediaTypeImage
	case videoExtensions[ext]:
		return MediaTypeVideo
	default:
		return ""
	}
}

type Upload struct {
	ID        string
	MediaType string
	Filename  string
	Path      string
	Size      int64
}

// SaveUpload stores an uploaded file under a fresh upload ID.
func (m *Manager) SaveUpload(src io.Reader, filename string) (*Upload, error) {
	mediaType := ClassifyMedia(filename)
	if mediaType == "" {
		return nil, fmt.Errorf("unsupported file type: %s", filepath.Ext(filename))
	}

	uploadID := uuid.New().String()
	dir := filepath.Join(m.UploadDir, uploadID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create upload dir: %w", err)
	}

	dest := filepath.Join(dir, "original"+strings.ToLower(filepath.Ext(filename)))
	f, err := os.Create(dest)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, io.LimitReader(src, m.MaxUploadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to write upload: %w", err)
	}

	return &Upload{ID: uploadID, MediaType: mediaType, Filename: filename, Path: dest, Size: written}, nil
}

// Job types processed by PixelForge. Each is registered against the existing
// GoTask handler registry.
const (
	JobImageMetadata  = "image.metadata"
	JobImageThumbnail = "image.thumbnail"
	JobImageResize    = "image.resize"
	JobImageCompress  = "image.compress"
	JobImageOptimize  = "image.optimize"

	JobVideoMetadata  = "video.metadata"
	JobVideoThumbnail = "video.thumbnail"
	JobVideoAudio     = "video.audio_extract"
	JobVideo1080p     = "video.transcode.1080p"
	JobVideo720p      = "video.transcode.720p"
	JobVideo480p      = "video.transcode.480p"
)

var labels = map[string]string{
	JobImageMetadata:  "Read metadata",
	JobImageThumbnail: "Generate thumbnail",
	JobImageResize:    "Resize 1280px",
	JobImageCompress:  "Compress",
	JobImageOptimize:  "Optimized version",
	JobVideoMetadata:  "Generate metadata",
	JobVideoThumbnail: "Generate thumbnail",
	JobVideoAudio:     "Extract audio",
	JobVideo1080p:     "Transcode 1080p",
	JobVideo720p:      "Transcode 720p",
	JobVideo480p:      "Transcode 480p",
}

// Label returns the human-readable name shown in the PixelForge UI.
func Label(jobType string) string {
	if l, ok := labels[jobType]; ok {
		return l
	}
	return jobType
}

// outputNames maps a job type to the file it produces in the output directory.
var outputNames = map[string]string{
	JobImageMetadata:  "metadata.json",
	JobImageThumbnail: "thumbnail.jpg",
	JobImageResize:    "resized.jpg",
	JobImageCompress:  "compressed.jpg",
	JobImageOptimize:  "optimized.jpg",
	JobVideoMetadata:  "metadata.json",
	JobVideoThumbnail: "thumbnail.jpg",
	JobVideoAudio:     "audio.m4a",
	JobVideo1080p:     "video_1080p.mp4",
	JobVideo720p:      "video_720p.mp4",
	JobVideo480p:      "video_480p.mp4",
}

// OutputFile is the deterministic output filename for a job type, shared by the
// handlers that write results and the API that links to them.
func OutputFile(jobType string) string { return outputNames[jobType] }

func (m *Manager) OutputPath(uploadID, jobType string) string {
	name := OutputFile(jobType)
	if name == "" {
		return ""
	}
	return filepath.Join(m.OutputDir, uploadID, name)
}

// Payload carried by every PixelForge job.
type Payload struct {
	UploadID   string `json:"upload_id"`
	SourcePath string `json:"source_path"`
	OutputPath string `json:"output_path"`
	Filename   string `json:"filename"`
	MediaType  string `json:"media_type"`
	Variant    string `json:"variant,omitempty"`
}

type JobSpec struct {
	Type     string
	Priority int
	Payload  json.RawMessage
}

// Pipelines: cheap jobs get higher priority so quick results surface first
// while heavy transcodes are still running.
var imagePipeline = []struct {
	Type     string
	Priority int
}{
	{JobImageMetadata, 10},
	{JobImageThumbnail, 8},
	{JobImageResize, 5},
	{JobImageCompress, 5},
	{JobImageOptimize, 4},
}

var videoPipeline = []struct {
	Type     string
	Priority int
	Variant  string
}{
	{JobVideoMetadata, 10, ""},
	{JobVideoThumbnail, 8, ""},
	{JobVideoAudio, 6, ""},
	{JobVideo480p, 5, "480p"},
	{JobVideo720p, 4, "720p"},
	{JobVideo1080p, 3, "1080p"},
}

// BuildJobSpecs expands one upload into the set of jobs that process it.
func (m *Manager) BuildJobSpecs(u *Upload) ([]JobSpec, error) {
	outDir := filepath.Join(m.OutputDir, u.ID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	build := func(jobType, variant string, priority int) (JobSpec, error) {
		payload := Payload{
			UploadID:   u.ID,
			SourcePath: u.Path,
			OutputPath: filepath.Join(outDir, OutputFile(jobType)),
			Filename:   u.Filename,
			MediaType:  u.MediaType,
			Variant:    variant,
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return JobSpec{}, err
		}
		return JobSpec{Type: jobType, Priority: priority, Payload: raw}, nil
	}

	specs := make([]JobSpec, 0, 6)
	if u.MediaType == MediaTypeImage {
		for _, step := range imagePipeline {
			spec, err := build(step.Type, "", step.Priority)
			if err != nil {
				return nil, err
			}
			specs = append(specs, spec)
		}
		return specs, nil
	}

	for _, step := range videoPipeline {
		spec, err := build(step.Type, step.Variant, step.Priority)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

func decodePayload(raw json.RawMessage) (*Payload, error) {
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("invalid media payload: %w", err)
	}
	if p.SourcePath == "" || p.OutputPath == "" {
		return nil, fmt.Errorf("media payload missing source or output path")
	}
	return &p, nil
}

func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
