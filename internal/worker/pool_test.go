package worker

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gotask/gotask/internal/jobs"
	"github.com/gotask/gotask/internal/logging"
	"github.com/gotask/gotask/internal/queue"
	"github.com/gotask/gotask/internal/storage"
)

func TestWorkerPoolExecution(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gotask-worker-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "worker_test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	q := queue.NewPriorityQueue(100)
	registry := jobs.NewRegistry()
	logger := logging.Setup("error") // quiet logs during test

	pool := NewWorkerPool(2, q, store, registry, logger)
	pool.Start()
	defer pool.Stop()

	ctx := context.Background()
	now := time.Now().UTC()

	// Create and enqueue echo job
	job := &jobs.Job{
		ID:          "job-echo-1",
		Type:        "echo",
		Payload:     []byte(`{"message":"hello world"}`),
		Priority:    1,
		Status:      jobs.StatusQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
		RunAt:       now,
		MaxAttempts: 3,
	}

	if err := store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("failed to enqueue job: %v", err)
	}

	// Wait for worker to process job
	var processedJob *jobs.Job
	for i := 0; i < 50; i++ {
		j, err := store.GetByID(ctx, "job-echo-1")
		if err == nil && j.Status == jobs.StatusSucceeded {
			processedJob = j
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if processedJob == nil {
		t.Fatalf("job was not processed to succeeded status in time")
	}

	if processedJob.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", processedJob.Attempts)
	}
	if processedJob.StartedAt == nil || processedJob.CompletedAt == nil {
		t.Errorf("expected started and completed timestamps to be set")
	}
}
