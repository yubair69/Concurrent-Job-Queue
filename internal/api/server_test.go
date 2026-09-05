package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gotask/gotask/internal/config"
	"github.com/gotask/gotask/internal/jobs"
	"github.com/gotask/gotask/internal/logging"
	"github.com/gotask/gotask/internal/queue"
	"github.com/gotask/gotask/internal/storage"
	"github.com/gotask/gotask/internal/worker"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "gotask-api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "api_test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	q := queue.NewPriorityQueue(100)
	registry := jobs.NewRegistry()
	logger := logging.Setup("error")
	pool := worker.NewWorkerPool(0, q, store, registry, logger)

	cfg := &config.Config{
		Port:            "0",
		DatabasePath:    dbPath,
		WorkerCount:     0,
		QueueCapacity:   100,
		DefaultRetries:  3,
		RetryBaseDelay:  1 * time.Second,
		RetryMaxDelay:   30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		LogLevel:        "error",
	}

	srv := NewServer(cfg, logger, store, q, registry, pool)
	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return srv, cleanup
}

func TestCreateJob(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := `{"type":"echo","payload":{"message":"hello"},"priority":5,"max_attempts":3}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["id"]; !ok {
		t.Error("expected response to contain id")
	}
	if resp["status"] != "queued" {
		t.Errorf("expected status queued, got %v", resp["status"])
	}
}

func TestCreateJobValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"type":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetJob(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	job, _ := jobs.NewJob("echo", []byte(`{}`), 1, 3)
	srv.store.Create(ctx, job)

	req := httptest.NewRequest(http.MethodGet, "/jobs/"+job.ID, nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp jobs.Job
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != job.ID {
		t.Errorf("expected job id %s, got %s", job.ID, resp.ID)
	}
}

func TestGetJobNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/jobs/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCancelJob(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	job, _ := jobs.NewJob("echo", []byte(`{}`), 1, 3)
	srv.store.Create(ctx, job)

	req := httptest.NewRequest(http.MethodDelete, "/jobs/"+job.ID, nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "cancelled" {
		t.Errorf("expected status cancelled, got %v", resp["status"])
	}
}

func TestMetrics(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var metrics map[string]any
	if err := json.NewDecoder(w.Body).Decode(&metrics); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := metrics["queued"]; !ok {
		t.Error("expected metrics to contain queued")
	}
	if _, ok := metrics["active_workers"]; !ok {
		t.Error("expected metrics to contain active_workers")
	}
}
