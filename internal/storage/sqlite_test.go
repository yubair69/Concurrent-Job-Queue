package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gotask/gotask/internal/jobs"
)

func TestSQLiteStore_LifecycleAndPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gotask-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	ctx := context.Background()

	// 1. Open store
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}

	now := time.Now().UTC()
	job := &jobs.Job{
		ID:          "job-123",
		Type:        "email",
		Payload:     []byte(`{"to":"test@example.com"}`),
		Priority:    5,
		Status:      jobs.StatusQueued,
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   now,
		UpdatedAt:   now,
		RunAt:       now,
	}

	// 2. Create job
	if err := store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// 3. Get job by ID
	retrieved, err := store.GetByID(ctx, "job-123")
	if err != nil {
		t.Fatalf("failed to get job by id: %v", err)
	}
	if retrieved.ID != job.ID || retrieved.Type != job.Type || retrieved.Priority != job.Priority || retrieved.Status != job.Status {
		t.Errorf("retrieved job mismatch: %+v", retrieved)
	}

	// 4. Update job status to running
	if err := retrieved.TransitionTo(jobs.StatusRunning); err != nil {
		t.Fatalf("failed to transition status: %v", err)
	}
	startTime := time.Now().UTC()
	retrieved.StartedAt = &startTime

	if err := store.Update(ctx, retrieved); err != nil {
		t.Fatalf("failed to update job: %v", err)
	}

	updated, err := store.GetByID(ctx, "job-123")
	if err != nil {
		t.Fatalf("failed to get updated job: %v", err)
	}
	if updated.Status != jobs.StatusRunning {
		t.Errorf("expected status running, got %s", updated.Status)
	}

	// 5. Close store and simulate restart
	if err := store.Close(); err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	store2, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer store2.Close()

	// 6. Test recovery of incomplete jobs (running -> queued)
	if err := store2.RecoverIncompleteJobs(ctx); err != nil {
		t.Fatalf("failed to recover incomplete jobs: %v", err)
	}

	recovered, err := store2.GetByID(ctx, "job-123")
	if err != nil {
		t.Fatalf("failed to get recovered job: %v", err)
	}
	if recovered.Status != jobs.StatusQueued {
		t.Errorf("expected recovered job status to be queued, got %s", recovered.Status)
	}

	// 7. Test GetNextQueued priority order
	job2 := &jobs.Job{
		ID:        "job-456",
		Type:      "echo",
		Payload:   []byte(`{}`),
		Priority:  10, // Higher priority than 5
		Status:    jobs.StatusQueued,
		CreatedAt: now,
		UpdatedAt: now,
		RunAt:     now,
	}
	if err := store2.Create(ctx, job2); err != nil {
		t.Fatalf("failed to create job2: %v", err)
	}

	next, err := store2.GetNextQueued(ctx)
	if err != nil {
		t.Fatalf("failed to get next queued job: %v", err)
	}
	if next.ID != "job-456" {
		t.Errorf("expected job-456 (higher priority), got %s", next.ID)
	}
}
