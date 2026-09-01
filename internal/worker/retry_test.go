package worker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gotask/gotask/internal/jobs"
	"github.com/gotask/gotask/internal/logging"
	"github.com/gotask/gotask/internal/queue"
	"github.com/gotask/gotask/internal/storage"
)

func TestWorkerRetryFlow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gotask-retry-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "retry_test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to store: %v", err)
	}
	defer store.Close()

	q := queue.NewPriorityQueue(100)
	registry := jobs.NewRegistry()
	logger := logging.Setup("error")

	var attemptCounter int32
	registry.Register("flaky", jobs.HandlerFunc(func(ctx context.Context, job *jobs.Job) error {
		current := atomic.AddInt32(&attemptCounter, 1)
		if current < 3 {
			return errors.New("temporary failure")
		}
		return nil // succeed on 3rd attempt
	}))

	pool := NewWorkerPool(1, q, store, registry, logger)
	// Use very short backoff for fast testing
	pool.SetRetryDelays(10*time.Millisecond, 50*time.Millisecond)
	pool.Start()
	defer pool.Stop()

	ctx := context.Background()
	now := time.Now().UTC()

	job := &jobs.Job{
		ID:          "job-retry-1",
		Type:        "flaky",
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

	// Wait for job to succeed after retries
	var finalJob *jobs.Job
	for i := 0; i < 100; i++ {
		j, err := store.GetByID(ctx, "job-retry-1")
		if err == nil && j.Status == jobs.StatusSucceeded {
			finalJob = j
			break
		}
		time.Sleep(30 * time.Millisecond)
	}

	if finalJob == nil {
		t.Fatalf("job did not succeed after retries")
	}

	if finalJob.Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", finalJob.Attempts)
	}
}
