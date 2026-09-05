package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gotask/gotask/internal/config"
	"github.com/gotask/gotask/internal/jobs"
	"github.com/gotask/gotask/internal/media"
	"github.com/gotask/gotask/internal/queue"
	"github.com/gotask/gotask/internal/storage"
	"github.com/gotask/gotask/internal/worker"
)

type Server struct {
	config     *config.Config
	server     *http.Server
	logger     *slog.Logger
	store      storage.Store
	queue      *queue.PriorityQueue
	registry   *jobs.Registry
	workerPool *worker.WorkerPool
	mux        *http.ServeMux
	media      *media.Manager
}

func NewServer(cfg *config.Config, logger *slog.Logger, store storage.Store, q *queue.PriorityQueue, registry *jobs.Registry, pool *worker.WorkerPool) *Server {
	mux := http.NewServeMux()

	s := &Server{
		config:     cfg,
		logger:     logger,
		store:      store,
		queue:      q,
		registry:   registry,
		workerPool: pool,
		mux:        mux,
	}

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /jobs", s.handleCreateJob)
	mux.HandleFunc("GET /jobs/", s.handleGetJob)
	mux.HandleFunc("DELETE /jobs/", s.handleCancelJob)
	mux.HandleFunc("GET /metrics", s.handleMetrics)

	s.server = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      s.loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server gracefully")
	return s.server.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	return s.server.Handler
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := fmt.Sprintf("req-%d", atomic.AddUint64(&requestIDCounter, 1))
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		r = r.WithContext(ctx)

		s.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "request_id", reqID)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggerWithContext(ctx context.Context) *slog.Logger {
	reqID, _ := ctx.Value(requestIDKey).(string)
	if reqID == "" {
		return s.logger
	}
	return s.logger.With("request_id", reqID)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) writeJSONError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

type CreateJobRequest struct {
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Priority    int             `json:"priority"`
	MaxAttempts int             `json:"max_attempts"`
}

const maxPayloadSize = 10 * 1024 * 1024 // 10 MB

type contextKey string

const requestIDKey contextKey = "request_id"

var requestIDCounter uint64

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)

	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	if req.Type == "" {
		s.writeJSONError(w, http.StatusBadRequest, "type is required")
		return
	}
	if req.Priority < 0 {
		req.Priority = 0
	}
	if req.MaxAttempts <= 0 {
		req.MaxAttempts = s.config.DefaultRetries
	}
	if req.Payload == nil {
		req.Payload = json.RawMessage("{}")
	}

	job, err := jobs.NewJob(req.Type, req.Payload, req.Priority, req.MaxAttempts)
	if err != nil {
		s.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	logger := s.loggerWithContext(ctx)
	if err := s.store.Create(ctx, job); err != nil {
		logger.Error("failed to create job", "job_id", job.ID, "error", err)
		s.writeJSONError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		logger.Error("failed to enqueue job", "job_id", job.ID, "error", err)
		s.writeJSONError(w, http.StatusInternalServerError, "failed to enqueue job")
		return
	}

	logger.Info("job created via API", "job_id", job.ID, "job_type", job.Type)
	s.writeJSON(w, http.StatusCreated, map[string]any{
		"id":           job.ID,
		"status":       string(job.Status),
		"type":         job.Type,
		"priority":     job.Priority,
		"max_attempts": job.MaxAttempts,
	})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/jobs/")
	if id == "" {
		s.writeJSONError(w, http.StatusBadRequest, "job id is required")
		return
	}

	logger := s.loggerWithContext(r.Context())
	job, err := s.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, jobs.ErrJobNotFound) {
			s.writeJSONError(w, http.StatusNotFound, "job not found")
			return
		}
		logger.Error("failed to get job", "job_id", id, "error", err)
		s.writeJSONError(w, http.StatusInternalServerError, "failed to get job")
		return
	}

	s.writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/jobs/")
	if id == "" {
		s.writeJSONError(w, http.StatusBadRequest, "job id is required")
		return
	}

	logger := s.loggerWithContext(r.Context())
	job, err := s.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, jobs.ErrJobNotFound) {
			s.writeJSONError(w, http.StatusNotFound, "job not found")
			return
		}
		logger.Error("failed to get job for cancellation", "job_id", id, "error", err)
		s.writeJSONError(w, http.StatusInternalServerError, "failed to cancel job")
		return
	}

	if job.Status != jobs.StatusQueued && job.Status != jobs.StatusRunning {
		s.writeJSONError(w, http.StatusConflict, "cannot cancel job in current state")
		return
	}

	wasRunning := job.Status == jobs.StatusRunning

	if err := job.TransitionTo(jobs.StatusCancelled); err != nil {
		logger.Error("invalid state transition", "job_id", id, "error", err)
		s.writeJSONError(w, http.StatusConflict, "cannot cancel job in current state")
		return
	}

	if err := s.store.Update(r.Context(), job); err != nil {
		logger.Error("failed to persist cancellation", "job_id", id, "error", err)
		s.writeJSONError(w, http.StatusInternalServerError, "failed to cancel job")
		return
	}

	if wasRunning && s.workerPool != nil {
		s.workerPool.CancelJob(id)
	}

	logger.Info("job cancelled via API", "job_id", id)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"id":     job.ID,
		"status": string(job.Status),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := s.loggerWithContext(ctx)
	counts, err := s.store.CountByStatus(ctx)
	if err != nil {
		logger.Error("failed to get metrics", "error", err)
		s.writeJSONError(w, http.StatusInternalServerError, "failed to retrieve metrics")
		return
	}

	totalProcessed := counts[jobs.StatusSucceeded] + counts[jobs.StatusFailed] + counts[jobs.StatusCancelled]
	activeWorkers := 0
	totalWorkers := 0
	totalProcessedByPool := int64(0)
	if s.workerPool != nil {
		poolStats := s.workerPool.Stats()
		activeWorkers = poolStats.ActiveWorkers
		totalWorkers = s.workerPool.WorkerCount()
		totalProcessedByPool = poolStats.TotalProcessed
	}

	queueStats := s.queue.Stats()

	metrics := map[string]any{
		"queued":          counts[jobs.StatusQueued],
		"running":         counts[jobs.StatusRunning],
		"succeeded":       counts[jobs.StatusSucceeded],
		"failed":          counts[jobs.StatusFailed],
		"cancelled":       counts[jobs.StatusCancelled],
		"active_workers":  activeWorkers,
		"total_workers":   totalWorkers,
		"total_processed": totalProcessed,
		"total_attempted": totalProcessedByPool,
		"queue_depth":     queueStats.Depth,
		"queue_enqueued":  queueStats.Enqueued,
		"queue_dequeued":  queueStats.Dequeued,
		"queue_capacity":  queueStats.Capacity,
	}

	s.writeJSON(w, http.StatusOK, metrics)
}
