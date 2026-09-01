package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Handler interface {
	Handle(ctx context.Context, job *Job) error
}

type HandlerFunc func(ctx context.Context, job *Job) error

func (f HandlerFunc) Handle(ctx context.Context, job *Job) error {
	return f(ctx, job)
}

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[string]Handler),
	}
	r.RegisterDefaults()
	return r
}

func (r *Registry) Register(jobType string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[jobType] = h
}

func (r *Registry) Get(jobType string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[jobType]
	return h, ok
}

func (r *Registry) RegisterDefaults() {
	// Echo handler
	r.Register("echo", HandlerFunc(func(ctx context.Context, job *Job) error {
		var payload map[string]any
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("invalid echo payload: %w", err)
		}
		return nil
	}))

	// Sleep handler (for testing concurrency, delay, cancellation)
	r.Register("sleep", HandlerFunc(func(ctx context.Context, job *Job) error {
		var payload struct {
			DurationMs int `json:"duration_ms"`
		}
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			// default to 100ms if payload invalid
			payload.DurationMs = 100
		}
		duration := time.Duration(payload.DurationMs) * time.Millisecond
		select {
		case <-time.After(duration):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}))

	// Email simulation handler
	r.Register("email", HandlerFunc(func(ctx context.Context, job *Job) error {
		var payload struct {
			To      string `json:"to"`
			Subject string `json:"subject"`
		}
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("invalid email payload: %w", err)
		}
		// Simulate work
		select {
		case <-time.After(50 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}))
}
