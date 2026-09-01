package queue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gotask/gotask/internal/jobs"
)

func TestPriorityAndFIFO(t *testing.T) {
	pq := NewPriorityQueue(10)
	ctx := context.Background()

	// Enqueue jobs with different priorities and same priority (FIFO check)
	j1 := &jobs.Job{ID: "low-1", Priority: 1}
	j2 := &jobs.Job{ID: "high-1", Priority: 10}
	j3 := &jobs.Job{ID: "low-2", Priority: 1} // Enqueued after low-1, same priority
	j4 := &jobs.Job{ID: "med-1", Priority: 5}

	pq.Enqueue(ctx, j1)
	pq.Enqueue(ctx, j2)
	pq.Enqueue(ctx, j3)
	pq.Enqueue(ctx, j4)

	// Expected order:
	// 1. high-1 (priority 10)
	// 2. med-1 (priority 5)
	// 3. low-1 (priority 1, sequence 1)
	// 4. low-2 (priority 1, sequence 3)

	expectedIDs := []string{"high-1", "med-1", "low-1", "low-2"}
	for i, expectedID := range expectedIDs {
		job, err := pq.Dequeue(ctx)
		if err != nil {
			t.Fatalf("failed to dequeue at index %d: %v", i, err)
		}
		if job.ID != expectedID {
			t.Errorf("at index %d: expected job ID %s, got %s", i, expectedID, job.ID)
		}
	}
}

func TestConcurrentProducersAndConsumers(t *testing.T) {
	pq := NewPriorityQueue(100)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	numProducers := 5
	jobsPerProducer := 50
	totalJobs := numProducers * jobsPerProducer

	var wgProducers sync.WaitGroup
	var wgConsumers sync.WaitGroup

	// Start producers
	for p := 0; p < numProducers; p++ {
		wgProducers.Add(1)
		go func(producerID int) {
			defer wgProducers.Done()
			for j := 0; j < jobsPerProducer; j++ {
				job := &jobs.Job{
					ID:       string(rune('A'+producerID)) + "-" + string(rune('0'+j)),
					Priority: j % 10,
				}
				_ = pq.Enqueue(ctx, job)
			}
		}(p)
	}

	// Start consumers (workers)
	numConsumers := 4
	var consumedTotal int64

	for c := 0; c < numConsumers; c++ {
		wgConsumers.Add(1)
		go func() {
			defer wgConsumers.Done()
			for {
				job, err := pq.Dequeue(ctx)
				if err != nil {
					if errorsIsClosed(err) || ctx.Err() != nil {
						return
					}
					return
				}
				if job != nil {
					val := atomic.AddInt64(&consumedTotal, 1)
					if val == int64(totalJobs) {
						pq.Close()
					}
				}
			}
		}()
	}

	wgProducers.Wait()
	// Wait a moment or wait for consumers to finish via Close
	wgConsumers.Wait()

	if consumedTotal != int64(totalJobs) {
		t.Errorf("expected %d consumed jobs, got %d", totalJobs, consumedTotal)
	}
}

func TestContextCancellation(t *testing.T) {
	pq := NewPriorityQueue(1) // capacity 1
	ctx := context.Background()

	// Fill queue to capacity
	j1 := &jobs.Job{ID: "j1", Priority: 1}
	if err := pq.Enqueue(ctx, j1); err != nil {
		t.Fatalf("failed to enqueue j1: %v", err)
	}

	// Try to enqueue second job with a canceled context (should block and then fail)
	cancelCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	j2 := &jobs.Job{ID: "j2", Priority: 1}
	err := pq.Enqueue(cancelCtx, j2)
	if err == nil {
		t.Error("expected error when enqueuing with canceled context on full queue, got nil")
	}

	// Test Dequeue with canceled context on empty queue
	pq2 := NewPriorityQueue(10)
	emptyCtx, emptyCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer emptyCancel()

	_, err = pq2.Dequeue(emptyCtx)
	if err == nil {
		t.Error("expected error when dequeuing with canceled context on empty queue, got nil")
	}
}

func errorsIsClosed(err error) bool {
	return err == ErrQueueClosed
}
