package cli

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gotask/gotask/internal/api"
	"github.com/gotask/gotask/internal/config"
	"github.com/gotask/gotask/internal/jobs"
	"github.com/gotask/gotask/internal/logging"
	"github.com/gotask/gotask/internal/queue"
	"github.com/gotask/gotask/internal/storage"
	"github.com/gotask/gotask/internal/worker"
)

func TestCLIIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gotask-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "cli_test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

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

	srv := api.NewServer(cfg, logger, store, q, registry, pool)
	testServer := httptest.NewServer(srv.Handler())
	defer testServer.Close()

	serverURL := testServer.URL

	// 1. Test submit
	argsSubmit := []string{"gotask", "submit", "--type", "echo", "--payload", `{"msg":"hello"}`, "--server", serverURL}
	code := Run(argsSubmit)
	if code != 0 {
		t.Errorf("expected submit exit code 0, got %d", code)
	}

	// Retrieve submitted job ID from store for status/cancel test
	ctx := t.Context()
	next, err := store.GetNextQueued(ctx)
	if err != nil || next == nil {
		t.Fatalf("failed to retrieve submitted job from store: %v", err)
	}
	jobID := next.ID

	// 2. Test status
	argsStatus := []string{"gotask", "status", "--server", serverURL, jobID}
	code = Run(argsStatus)
	if code != 0 {
		t.Errorf("expected status exit code 0, got %d", code)
	}

	// 3. Test workers (metrics)
	argsWorkers := []string{"gotask", "workers", "--server", serverURL}
	code = Run(argsWorkers)
	if code != 0 {
		t.Errorf("expected workers exit code 0, got %d", code)
	}

	// 4. Test cancel
	argsCancel := []string{"gotask", "cancel", "--server", serverURL, jobID}
	code = Run(argsCancel)
	if code != 0 {
		t.Errorf("expected cancel exit code 0, got %d", code)
	}
}
