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

func TestWorkerPoolCancelRunningJob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gotask-cancel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "cancel_test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	q := queue.NewPriorityQueue(100)
	registry := jobs.NewRegistry()
	logger := logging.Setup("error")

	registry.Register("block", jobs.HandlerFunc(func(ctx context.Context, job *jobs.Job) error {
		<-ctx.Done()
		return ctx.Err()
	}))

	pool := NewWorkerPool(1, q, store, registry, logger)
	pool.Start()
	defer pool.Stop()

	ctx := context.Background()
	now := time.Now().UTC()

	job := &jobs.Job{
		ID:          "job-block-1",
		Type:        "block",
		Payload:     []byte(`{}`),
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

	// Wait for job to start running
	var runningJob *jobs.Job
	for i := 0; i < 50; i++ {
		j, err := store.GetByID(ctx, "job-block-1")
		if err == nil && j.Status == jobs.StatusRunning {
			runningJob = j
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if runningJob == nil {
		t.Fatalf("job did not start running in time")
	}

	// Cancel the running job
	pool.CancelJob("job-block-1")

	// Wait for job to be marked cancelled
	var cancelledJob *jobs.Job
	for i := 0; i < 50; i++ {
		j, err := store.GetByID(ctx, "job-block-1")
		if err == nil && j.Status == jobs.StatusCancelled {
			cancelledJob = j
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if cancelledJob == nil {
		t.Fatalf("job was not cancelled in time")
	}

	if cancelledJob.CompletedAt == nil {
		t.Errorf("expected completed_at to be set for cancelled job")
	}
}

func TestWorkerPoolGracefulShutdown(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gotask-shutdown-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "shutdown_test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	q := queue.NewPriorityQueue(100)
	registry := jobs.NewRegistry()
	logger := logging.Setup("error")

	pool := NewWorkerPool(2, q, store, registry, logger)
	pool.Start()

	ctx := context.Background()
	now := time.Now().UTC()

	job := &jobs.Job{
		ID:          "job-shutdown-1",
		Type:        "echo",
		Payload:     []byte(`{}`),
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

	// Give workers a moment to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown should complete without hanging or leaking
	pool.Stop()

	// Verify job state is persisted
	j, err := store.GetByID(ctx, "job-shutdown-1")
	if err != nil {
		t.Fatalf("failed to get job after shutdown: %v", err)
	}
	if j.Status != jobs.StatusSucceeded && j.Status != jobs.StatusQueued {
		t.Errorf("unexpected job status after shutdown: %s", j.Status)
	}
}
