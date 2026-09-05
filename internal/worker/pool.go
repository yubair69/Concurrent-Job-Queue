package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
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
	runningJobs sync.Map
	activeCount int64
	processed   int64
	slots       []workerSlot
	ctx         context.Context
	cancel      context.CancelFunc
}

type PoolStats struct {
	ActiveWorkers  int
	TotalProcessed int64
}

// workerSlot holds the live state of a single worker goroutine. current is nil
// while the worker is idle; each slot is only written by its own worker.
type workerSlot struct {
	current   atomic.Pointer[currentJob]
	processed atomic.Int64
}

type currentJob struct {
	jobID   string
	jobType string
	since   time.Time
}

type WorkerState struct {
	ID             int        `json:"id"`
	Status         string     `json:"status"`
	JobID          string     `json:"job_id,omitempty"`
	JobType        string     `json:"job_type,omitempty"`
	Since          *time.Time `json:"since,omitempty"`
	ProcessedCount int64      `json:"processed_count"`
}

// WorkerStates returns a snapshot of what every worker is doing right now.
func (wp *WorkerPool) WorkerStates() []WorkerState {
	states := make([]WorkerState, 0, len(wp.slots))
	for i := range wp.slots {
		state := WorkerState{
			ID:             i + 1,
			Status:         "idle",
			ProcessedCount: wp.slots[i].processed.Load(),
		}
		if cur := wp.slots[i].current.Load(); cur != nil {
			since := cur.since
			state.Status = "busy"
			state.JobID = cur.jobID
			state.JobType = cur.jobType
			state.Since = &since
		}
		states = append(states, state)
	}
	return states
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
		slots:       make([]workerSlot, workerCount),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (wp *WorkerPool) WorkerCount() int {
	return wp.workerCount
}

func (wp *WorkerPool) Stats() PoolStats {
	return PoolStats{
		ActiveWorkers:  int(atomic.LoadInt64(&wp.activeCount)),
		TotalProcessed: atomic.LoadInt64(&wp.processed),
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

func (wp *WorkerPool) CancelJob(jobID string) {
	if val, ok := wp.runningJobs.Load(jobID); ok {
		if cancel, ok := val.(context.CancelFunc); ok {
			cancel()
		}
	}
}

func (wp *WorkerPool) Stop() {
	wp.logger.Info("stopping worker pool")

	wp.runningJobs.Range(func(key, value any) bool {
		if cancel, ok := value.(context.CancelFunc); ok {
			cancel()
		}
		return true
	})

	wp.cancel()
	wp.queue.Close()
	wp.wg.Wait()
	wp.logger.Info("worker pool stopped successfully")
}

func (wp *WorkerPool) Drain() {
	wp.queue.Close()
	wp.wg.Wait()
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

		slot := &wp.slots[workerID-1]
		slot.current.Store(&currentJob{jobID: job.ID, jobType: job.Type, since: time.Now().UTC()})
		wp.executeJob(workerID, job, logger)
		slot.current.Store(nil)
		slot.processed.Add(1)
	}
}

func (wp *WorkerPool) executeJob(workerID int, job *jobs.Job, logger *slog.Logger) {
	atomic.AddInt64(&wp.activeCount, 1)
	defer atomic.AddInt64(&wp.activeCount, -1)

	atomic.AddInt64(&wp.processed, 1)

	startTime := time.Now().UTC()
	job.Attempts++
	job.StartedAt = &startTime

	jobCtx, jobCancel := context.WithCancel(wp.ctx)
	wp.runningJobs.Store(job.ID, jobCancel)
	defer wp.runningJobs.Delete(job.ID)
	defer jobCancel()

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
		execErr = handler.Handle(jobCtx, job)
	}

	completedTime := time.Now().UTC()
	duration := completedTime.Sub(startTime)

	if jobCtx.Err() != nil {
		if reloaded, err := wp.store.GetByID(wp.ctx, job.ID); err == nil {
			reloaded.CompletedAt = &completedTime
			reloaded.LastError = "cancelled"
			reloaded.Status = jobs.StatusCancelled
			_ = wp.store.Update(wp.ctx, reloaded)
		}
		job.LastError = "cancelled"
		logger.Info("job cancelled", "job_id", job.ID, "job_type", job.Type, "duration_ms", duration.Milliseconds(), "worker_id", workerID)
		return
	}

	if execErr != nil {
		job.LastError = execErr.Error()
		isPerm := jobs.IsPermanent(execErr)
		isMaxAttempts := job.Attempts >= job.MaxAttempts

		if isPerm || isMaxAttempts {
			job.CompletedAt = &completedTime
			if err := job.TransitionTo(jobs.StatusFailed); err != nil {
				logger.Error("failed to transition job to failed", "job_id", job.ID, "error", err)
			}
			logger.Error("job permanently failed", "job_id", job.ID, "job_type", job.Type, "attempts", job.Attempts, "duration_ms", duration.Milliseconds(), "worker_id", workerID, "error", execErr)
		} else {
			delay := jobs.CalculateBackoff(job.Attempts, wp.baseDelay, wp.maxDelay)
			job.RunAt = time.Now().UTC().Add(delay)

			if err := job.TransitionTo(jobs.StatusFailed); err != nil {
				logger.Error("failed to transition retryable job to failed", "job_id", job.ID, "error", err)
			}
			if err := job.TransitionTo(jobs.StatusQueued); err != nil {
				logger.Error("failed to transition retryable job to queued", "job_id", job.ID, "error", err)
			}

			logger.Warn("job failed, scheduling retry", "job_id", job.ID, "attempt", job.Attempts, "max_attempts", job.MaxAttempts, "delay_ms", delay.Milliseconds(), "error", execErr)

			if err := wp.store.Update(wp.ctx, job); err != nil {
				logger.Error("failed to persist retry state", "job_id", job.ID, "error", err)
				return
			}

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
		logger.Info("job succeeded", "job_id", job.ID, "job_type", job.Type, "duration_ms", duration.Milliseconds(), "worker_id", workerID)
	}

	if err := wp.store.Update(wp.ctx, job); err != nil {
		logger.Error("failed to persist completed job state", "job_id", job.ID, "error", err)
	}
}
