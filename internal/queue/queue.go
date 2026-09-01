package queue

import (
	"container/heap"
	"context"
	"errors"
	"sync"

	"github.com/gotask/gotask/internal/jobs"
)

var (
	ErrQueueClosed  = errors.New("queue is closed")
	ErrQueueFull    = errors.New("queue is at maximum capacity")
	ErrQueueTimeout = errors.New("queue operation timed out")
)

type item struct {
	job      *jobs.Job
	index    int
	sequence int64
}

// priorityHeap implements container/heap.Interface
type priorityHeap []*item

func (h priorityHeap) Len() int { return len(h) }

func (h priorityHeap) Less(i, j int) bool {
	// Higher priority comes first
	if h[i].job.Priority != h[j].job.Priority {
		return h[i].job.Priority > h[j].job.Priority
	}
	// If priority is equal, FIFO (lower sequence number came first)
	return h[i].sequence < h[j].sequence
}

func (h priorityHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *priorityHeap) Push(x any) {
	n := len(*h)
	it := x.(*item)
	it.index = n
	*h = append(*h, it)
}

func (h *priorityHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	old[n-1] = nil // avoid memory leak
	it.index = -1
	*h = old[0 : n-1]
	return it
}

type PriorityQueue struct {
	mu       sync.Mutex
	cond     *sync.Cond
	items    priorityHeap
	capacity int
	seq      int64
	closed   bool
}

func NewPriorityQueue(capacity int) *PriorityQueue {
	pq := &PriorityQueue{
		items:    make(priorityHeap, 0),
		capacity: capacity,
	}
	pq.cond = sync.NewCond(&pq.mu)
	heap.Init(&pq.items)
	return pq
}

func (pq *PriorityQueue) Enqueue(ctx context.Context, job *jobs.Job) error {
	if job == nil {
		return errors.New("cannot enqueue nil job")
	}

	pq.mu.Lock()
	defer pq.mu.Unlock()

	for pq.capacity > 0 && len(pq.items) >= pq.capacity {
		if pq.closed {
			return ErrQueueClosed
		}

		// Wait for capacity with context cancellation support
		done := make(chan struct{})

		go func() {
			pq.mu.Lock()
			defer pq.mu.Unlock()
			for len(pq.items) >= pq.capacity && !pq.closed {
				pq.cond.Wait()
			}
			close(done)
		}()

		// Wait either for condition signal or context cancellation
		pq.mu.Unlock()
		select {
		case <-ctx.Done():
			pq.mu.Lock()
			pq.cond.Broadcast() // wake up goroutine
			return ctx.Err()
		case <-done:
			pq.mu.Lock()
			if pq.closed {
				return ErrQueueClosed
			}
		}
	}

	if pq.closed {
		return ErrQueueClosed
	}

	pq.seq++
	it := &item{
		job:      job,
		sequence: pq.seq,
	}
	heap.Push(&pq.items, it)
	pq.cond.Signal()

	return nil
}

func (pq *PriorityQueue) Dequeue(ctx context.Context) (*jobs.Job, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for len(pq.items) == 0 {
		if pq.closed {
			return nil, ErrQueueClosed
		}

		done := make(chan struct{})
		go func() {
			pq.mu.Lock()
			defer pq.mu.Unlock()
			for len(pq.items) == 0 && !pq.closed {
				pq.cond.Wait()
			}
			close(done)
		}()

		pq.mu.Unlock()
		select {
		case <-ctx.Done():
			pq.mu.Lock()
			pq.cond.Broadcast()
			return nil, ctx.Err()
		case <-done:
			pq.mu.Lock()
			if pq.closed && len(pq.items) == 0 {
				return nil, ErrQueueClosed
			}
		}
	}

	if pq.closed && len(pq.items) == 0 {
		return nil, ErrQueueClosed
	}

	it := heap.Pop(&pq.items).(*item)
	pq.cond.Signal() // Signal producers that capacity is available

	return it.job, nil
}

func (pq *PriorityQueue) Close() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.closed = true
	pq.cond.Broadcast()
}

func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.items)
}

func (pq *PriorityQueue) Capacity() int {
	return pq.capacity
}
