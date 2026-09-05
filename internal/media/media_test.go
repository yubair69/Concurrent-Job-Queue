package media

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotask/gotask/internal/jobs"
)

func TestClassifyMedia(t *testing.T) {
	cases := map[string]string{
		"photo.JPG": MediaTypeImage,
		"clip.mp4":  MediaTypeVideo,
		"notes.txt": "",
	}
	for filename, want := range cases {
		if got := ClassifyMedia(filename); got != want {
			t.Errorf("ClassifyMedia(%q) = %q, want %q", filename, got, want)
		}
	}
}

func TestBuildJobSpecsCoversPipeline(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "uploads"), filepath.Join(dir, "outputs"), 10)

	specs, err := m.BuildJobSpecs(&Upload{ID: "u1", MediaType: MediaTypeImage, Filename: "a.png", Path: "a.png"})
	if err != nil {
		t.Fatalf("BuildJobSpecs: %v", err)
	}
	if len(specs) != len(imagePipeline) {
		t.Fatalf("got %d image jobs, want %d", len(specs), len(imagePipeline))
	}

	videoSpecs, err := m.BuildJobSpecs(&Upload{ID: "u2", MediaType: MediaTypeVideo, Filename: "a.mp4", Path: "a.mp4"})
	if err != nil {
		t.Fatalf("BuildJobSpecs video: %v", err)
	}
	if len(videoSpecs) != len(videoPipeline) {
		t.Fatalf("got %d video jobs, want %d", len(videoSpecs), len(videoPipeline))
	}
}

// The image pipeline must do real work: decode a source file and write a
// genuinely smaller derived image.
func TestImageThumbnailProducesRealOutput(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.png")

	img := image.NewRGBA(image.Rect(0, 0, 900, 600))
	for x := 0; x < 900; x++ {
		for y := 0; y < 600; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 120, A: 255})
		}
	}
	f, err := os.Create(src)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode source: %v", err)
	}
	f.Close()

	out := filepath.Join(dir, "thumbnail.jpg")
	payload, _ := json.Marshal(Payload{SourcePath: src, OutputPath: out, Filename: "source.png", MediaType: MediaTypeImage})

	handler := imageResizeHandler(320, 80)
	if err := handler(context.Background(), &jobs.Job{Payload: payload}); err != nil {
		t.Fatalf("thumbnail handler: %v", err)
	}

	result, err := os.Open(out)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	defer result.Close()

	cfg, _, err := image.DecodeConfig(result)
	if err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if cfg.Width != 320 {
		t.Errorf("thumbnail width = %d, want 320", cfg.Width)
	}
	if cfg.Height >= 600 {
		t.Errorf("thumbnail height = %d, expected scaled down", cfg.Height)
	}
}

func TestImageHandlerRejectsBadSourcePermanently(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "broken.png")
	if err := os.WriteFile(src, []byte("not an image"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	payload, _ := json.Marshal(Payload{SourcePath: src, OutputPath: filepath.Join(dir, "out.jpg")})
	err := imageResizeHandler(320, 80)(context.Background(), &jobs.Job{Payload: payload})
	if err == nil {
		t.Fatal("expected an error for an undecodable image")
	}
	if !jobs.IsPermanent(err) {
		t.Errorf("expected a permanent error so the job is not retried, got %v", err)
	}
}
