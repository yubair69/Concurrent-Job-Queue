package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gotask/gotask/internal/jobs"
	"github.com/gotask/gotask/internal/queue"
	"github.com/gotask/gotask/internal/storage"
)

type WorkerPool struct {
	workerCount int
	queue       *queue.PriorityQueue
	store       storage.Store
	registry    *jobs.Registry
	logger      *slog.Logger
	baseDelay   time.Duration
	maxDelay    time.Duration
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewWorkerPool(workerCount int, q *queue.PriorityQueue, s storage.Store, r *jobs.Registry, logger *slog.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workerCount: workerCount,
		queue:       q,
		store:       s,
		registry:    r,
		logger:      logger,
		baseDelay:   1 * time.Second,
		maxDelay:    30 * time.Second,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (wp *WorkerPool) SetRetryDelays(base, max time.Duration) {
	wp.baseDelay = base
	wp.maxDelay = max
}

func (wp *WorkerPool) Start() {
	wp.logger.Info("starting worker pool", "workers", wp.workerCount)
	for i := 1; i <= wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.runWorker(i)
	}
}

func (wp *WorkerPool) Stop() {
	wp.logger.Info("stopping worker pool")
	wp.cancel()
	wp.queue.Close()
	wp.wg.Wait()
	wp.logger.Info("worker pool stopped successfully")
}

func (wp *WorkerPool) runWorker(workerID int) {
	defer wp.wg.Done()
	logger := wp.logger.With("worker_id", workerID)
	logger.Debug("worker started")

	for {
		job, err := wp.queue.Dequeue(wp.ctx)
		if err != nil {
			if wp.ctx.Err() != nil {
				logger.Debug("worker stopping due to context cancellation")
				return
			}
			logger.Debug("queue closed, worker exiting", "error", err)
			return
		}

		if job == nil {
			continue
		}

		wp.executeJob(workerID, job, logger)
	}
}

func (wp *WorkerPool) executeJob(workerID int, job *jobs.Job, logger *slog.Logger) {
	startTime := time.Now().UTC()
	job.Attempts++
	job.StartedAt = &startTime

	// Transition to running
	if err := job.TransitionTo(jobs.StatusRunning); err != nil {
		logger.Error("failed to transition job to running", "job_id", job.ID, "error", err)
		return
	}

	if err := wp.store.Update(wp.ctx, job); err != nil {
		logger.Error("failed to persist running job state", "job_id", job.ID, "error", err)
		return
	}

	logger.Info("executing job", "job_id", job.ID, "job_type", job.Type, "attempt", job.Attempts)

	handler, exists := wp.registry.Get(job.Type)
	var execErr error
	if !exists {
		execErr = fmt.Errorf("no handler registered for job type: %s", job.Type)
	} else {
		execErr = handler.Handle(wp.ctx, job)
	}

	completedTime := time.Now().UTC()
	duration := completedTime.Sub(startTime)

	if execErr != nil {
		job.LastError = execErr.Error()
		isPerm := jobs.IsPermanent(execErr)
		isMaxAttempts := job.Attempts >= job.MaxAttempts

		if isPerm || isMaxAttempts {
			// Final failure
			job.CompletedAt = &completedTime
			if err := job.TransitionTo(jobs.StatusFailed); err != nil {
				logger.Error("failed to transition job to failed", "job_id", job.ID, "error", err)
			}
			logger.Error("job permanently failed", "job_id", job.ID, "job_type", job.Type, "attempts", job.Attempts, "duration_ms", duration.Milliseconds(), "error", execErr)
		} else {
			// Retryable failure
			delay := jobs.CalculateBackoff(job.Attempts, wp.baseDelay, wp.maxDelay)
			job.RunAt = time.Now().UTC().Add(delay)

			// Transition running -> failed -> queued
			if err := job.TransitionTo(jobs.StatusFailed); err != nil {
				logger.Error("failed to transition retryable job to failed", "job_id", job.ID, "error", err)
			}
			if err := job.TransitionTo(jobs.StatusQueued); err != nil {
				logger.Error("failed to transition retryable job to queued", "job_id", job.ID, "error", err)
			}

			logger.Warn("job failed, scheduling retry", "job_id", job.ID, "attempt", job.Attempts, "max_attempts", job.MaxAttempts, "delay_ms", delay.Milliseconds(), "error", execErr)

			// Persist state in store before scheduling re-enqueue
			if err := wp.store.Update(wp.ctx, job); err != nil {
				logger.Error("failed to persist retry state", "job_id", job.ID, "error", err)
				return
			}

			// Schedule re-enqueue after backoff delay
			time.AfterFunc(delay, func() {
				if wp.ctx.Err() == nil {
					_ = wp.queue.Enqueue(context.Background(), job)
				}
			})
			return
		}
	} else {
		job.CompletedAt = &completedTime
		if err := job.TransitionTo(jobs.StatusSucceeded); err != nil {
			logger.Error("failed to transition job to succeeded", "job_id", job.ID, "error", err)
		}
		logger.Info("job succeeded", "job_id", job.ID, "job_type", job.Type, "duration_ms", duration.Milliseconds())
	}

	if err := wp.store.Update(wp.ctx, job); err != nil {
		logger.Error("failed to persist completed job state", "job_id", job.ID, "error", err)
	}
}
