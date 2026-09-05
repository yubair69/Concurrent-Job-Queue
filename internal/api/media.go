package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gotask/gotask/internal/jobs"
	"github.com/gotask/gotask/internal/media"
)

// RegisterMediaRoutes adds the PixelForge endpoints to the existing server.
// Called once after NewServer and before Start; the core GoTask routes are
// untouched.
func (s *Server) RegisterMediaRoutes(mgr *media.Manager) {
	s.media = mgr

	s.mux.HandleFunc("POST /api/uploads", s.handleCreateUpload)
	s.mux.HandleFunc("GET /api/uploads", s.handleListUploads)
	s.mux.HandleFunc("GET /api/uploads/", s.handleGetUpload)
	s.mux.HandleFunc("GET /api/jobs", s.handleListJobs)
	s.mux.HandleFunc("GET /api/workers", s.handleWorkers)
	s.mux.HandleFunc("GET /api/outputs/", s.handleServeOutput)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/metrics", s.handleMetrics)

	if s.config.StaticDir != "" {
		if _, err := os.Stat(s.config.StaticDir); err == nil {
			s.mux.HandleFunc("GET /", s.handleStatic)
		}
	}
}

type jobView struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	Attempts   int    `json:"attempts"`
	Error      string `json:"error,omitempty"`
	OutputURL  string `json:"output_url,omitempty"`
	SizeBytes  int64  `json:"size_bytes,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

type uploadView struct {
	UploadID   string    `json:"upload_id"`
	Filename   string    `json:"filename"`
	MediaType  string    `json:"media_type"`
	Status     string    `json:"status"`
	Jobs       []jobView `json:"jobs"`
	QueueDepth int       `json:"queue_depth"`
	Workers    any       `json:"workers,omitempty"`
	Engine     string    `json:"engine"`
}

func (s *Server) handleCreateUpload(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerWithContext(r.Context())

	maxBytes := s.media.MaxUploadBytes
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		s.writeJSONError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.writeJSONError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	upload, err := s.media.SaveUpload(file, header.Filename)
	if err != nil {
		s.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	specs, err := s.media.BuildJobSpecs(upload)
	if err != nil {
		logger.Error("failed to build job specs", "upload_id", upload.ID, "error", err)
		s.writeJSONError(w, http.StatusInternalServerError, "failed to plan processing jobs")
		return
	}

	ctx := r.Context()
	created := make([]jobView, 0, len(specs))
	for _, spec := range specs {
		job, err := jobs.NewJob(spec.Type, spec.Payload, spec.Priority, s.config.DefaultRetries)
		if err != nil {
			logger.Error("failed to build job", "error", err)
			s.writeJSONError(w, http.StatusInternalServerError, "failed to create job")
			return
		}
		job.GroupID = upload.ID

		if err := s.store.Create(ctx, job); err != nil {
			logger.Error("failed to persist job", "job_id", job.ID, "error", err)
			s.writeJSONError(w, http.StatusInternalServerError, "failed to create job")
			return
		}
		if err := s.queue.Enqueue(ctx, job); err != nil {
			logger.Error("failed to enqueue job", "job_id", job.ID, "error", err)
			s.writeJSONError(w, http.StatusInternalServerError, "failed to enqueue job")
			return
		}
		created = append(created, jobView{
			ID:     job.ID,
			Type:   job.Type,
			Label:  media.Label(job.Type),
			Status: string(job.Status),
		})
	}

	logger.Info("upload accepted", "upload_id", upload.ID, "media_type", upload.MediaType, "jobs", len(created))
	s.writeJSON(w, http.StatusCreated, uploadView{
		UploadID:  upload.ID,
		Filename:  upload.Filename,
		MediaType: upload.MediaType,
		Status:    "queued",
		Jobs:      created,
		Engine:    s.engineName(upload.MediaType),
	})
}

func (s *Server) engineName(mediaType string) string {
	if mediaType == media.MediaTypeVideo && !s.media.HasFFmpeg() {
		return "simulated"
	}
	if mediaType == media.MediaTypeVideo {
		return "ffmpeg"
	}
	return "go-image"
}

func (s *Server) buildUploadView(ctx context.Context, uploadID string) (*uploadView, error) {
	groupJobs, err := s.store.ListByGroupID(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	if len(groupJobs) == 0 {
		return nil, jobs.ErrJobNotFound
	}

	view := &uploadView{UploadID: uploadID, Jobs: make([]jobView, 0, len(groupJobs))}

	var succeeded, failed, running int
	for _, job := range groupJobs {
		var payload media.Payload
		_ = json.Unmarshal(job.Payload, &payload)
		if view.Filename == "" {
			view.Filename = payload.Filename
			view.MediaType = payload.MediaType
		}

		jv := jobView{
			ID:       job.ID,
			Type:     job.Type,
			Label:    media.Label(job.Type),
			Status:   string(job.Status),
			Attempts: job.Attempts,
			Error:    job.LastError,
		}
		if job.StartedAt != nil && job.CompletedAt != nil {
			jv.DurationMs = job.CompletedAt.Sub(*job.StartedAt).Milliseconds()
		}

		switch job.Status {
		case jobs.StatusSucceeded:
			succeeded++
			if name := media.OutputFile(job.Type); name != "" {
				if stat, err := os.Stat(s.media.OutputPath(uploadID, job.Type)); err == nil {
					jv.OutputURL = "/api/outputs/" + uploadID + "/" + name
					jv.SizeBytes = stat.Size()
				}
			}
		case jobs.StatusFailed:
			failed++
		case jobs.StatusRunning:
			running++
		}
		view.Jobs = append(view.Jobs, jv)
	}

	switch {
	case succeeded+failed == len(groupJobs) && failed > 0:
		view.Status = "completed_with_failures"
	case succeeded == len(groupJobs):
		view.Status = "completed"
	case running > 0:
		view.Status = "processing"
	default:
		view.Status = "queued"
	}

	view.QueueDepth = s.queue.Stats().Depth
	if s.workerPool != nil {
		view.Workers = s.workerPool.WorkerStates()
	}
	view.Engine = s.engineName(view.MediaType)
	return view, nil
}

func (s *Server) handleGetUpload(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/uploads/"), "/")
	if id == "" {
		s.writeJSONError(w, http.StatusBadRequest, "upload id is required")
		return
	}

	view, err := s.buildUploadView(r.Context(), id)
	if err != nil {
		if err == jobs.ErrJobNotFound {
			s.writeJSONError(w, http.StatusNotFound, "upload not found")
			return
		}
		s.loggerWithContext(r.Context()).Error("failed to load upload", "upload_id", id, "error", err)
		s.writeJSONError(w, http.StatusInternalServerError, "failed to load upload")
		return
	}
	s.writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleListUploads(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 20)
	ids, err := s.store.ListGroupIDs(r.Context(), limit)
	if err != nil {
		s.writeJSONError(w, http.StatusInternalServerError, "failed to list uploads")
		return
	}

	uploads := make([]*uploadView, 0, len(ids))
	for _, id := range ids {
		view, err := s.buildUploadView(r.Context(), id)
		if err != nil {
			continue
		}
		uploads = append(uploads, view)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"uploads": uploads})
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)
	recent, err := s.store.ListRecent(r.Context(), limit)
	if err != nil {
		s.writeJSONError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"jobs": recent})
}

func (s *Server) handleWorkers(w http.ResponseWriter, r *http.Request) {
	if s.workerPool == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{"workers": []any{}})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"workers":     s.workerPool.WorkerStates(),
		"queue_depth": s.queue.Stats().Depth,
	})
}

// handleServeOutput serves processed files, confined to the output directory.
func (s *Server) handleServeOutput(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(r.URL.Path, "/api/outputs/")
	parts := strings.Split(rel, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		s.writeJSONError(w, http.StatusBadRequest, "invalid output path")
		return
	}

	uploadID := filepath.Base(parts[0])
	name := filepath.Base(parts[1])
	path := filepath.Join(s.media.OutputDir, uploadID, name)

	root, err := filepath.Abs(s.media.OutputDir)
	if err != nil {
		s.writeJSONError(w, http.StatusInternalServerError, "failed to resolve output path")
		return
	}
	abs, err := filepath.Abs(path)
	if err != nil || !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		s.writeJSONError(w, http.StatusBadRequest, "invalid output path")
		return
	}
	if _, err := os.Stat(abs); err != nil {
		s.writeJSONError(w, http.StatusNotFound, "output not found")
		return
	}
	http.ServeFile(w, r, abs)
}

// handleStatic serves the built frontend with SPA fallback.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if clean == "." || clean == string(os.PathSeparator) {
		clean = "index.html"
	}

	path := filepath.Join(s.config.StaticDir, clean)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.config.StaticDir, "index.html"))
}

func queryInt(r *http.Request, key string, fallback int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}
