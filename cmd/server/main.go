package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gotask/gotask/internal/api"
	"github.com/gotask/gotask/internal/config"
	"github.com/gotask/gotask/internal/jobs"
	"github.com/gotask/gotask/internal/logging"
	"github.com/gotask/gotask/internal/media"
	"github.com/gotask/gotask/internal/queue"
	"github.com/gotask/gotask/internal/storage"
	"github.com/gotask/gotask/internal/worker"
)

func main() {
	cfg := config.Load()
	logger := logging.Setup(cfg.LogLevel)

	logger.Info("initializing gotask server", "port", cfg.Port, "db", cfg.DatabasePath)

	store, err := storage.NewSQLiteStore(cfg.DatabasePath)
	if err != nil {
		logger.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.RecoverIncompleteJobs(context.Background()); err != nil {
		logger.Error("failed to recover incomplete jobs", "error", err)
	}

	mediaManager := media.NewManager(cfg.UploadDir, cfg.OutputDir, cfg.MaxUploadMB)
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		logger.Error("failed to create upload dir", "error", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		logger.Error("failed to create output dir", "error", err)
		os.Exit(1)
	}

	q := queue.NewPriorityQueue(cfg.QueueCapacity)
	registry := jobs.NewRegistry()
	mediaManager.Register(registry)
	logger.Info("pixelforge media engine ready", "ffmpeg", mediaManager.HasFFmpeg(),
		"uploads", cfg.UploadDir, "outputs", cfg.OutputDir)
	wp := worker.NewWorkerPool(cfg.WorkerCount, q, store, registry, logger)
	wp.SetRetryDelays(cfg.RetryBaseDelay, cfg.RetryMaxDelay)
	wp.Start()

	srv := api.NewServer(cfg, logger, store, q, registry, wp)
	srv.RegisterMediaRoutes(mediaManager)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
		}
	}()

	sig := <-sigChan
	logger.Info("received shutdown signal", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	q.Close()

	done := make(chan struct{})
	go func() {
		wp.Drain()
		close(done)
	}()

	select {
	case <-done:
		// graceful drain completed
	case <-ctx.Done():
		logger.Warn("shutdown timeout exceeded, cancelling remaining jobs")
		wp.Stop()
		<-done
	}

	store.Close()
	logger.Info("gotask server stopped gracefully")
}
