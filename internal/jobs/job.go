package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type JobStatus string

const (
	StatusQueued    JobStatus = "queued"
	StatusRunning   JobStatus = "running"
	StatusSucceeded JobStatus = "succeeded"
	StatusFailed    JobStatus = "failed"
	StatusCancelled JobStatus = "cancelled"
)

type Job struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Priority    int             `json:"priority"`
	Status      JobStatus       `json:"status"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	RunAt       time.Time       `json:"run_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	LastError   string          `json:"last_error,omitempty"`
}

var (
	ErrInvalidTransition = errors.New("invalid job state transition")
	ErrJobNotFound       = errors.New("job not found")
)

// IsValidTransition checks whether transitioning from current status to target status is allowed.
func IsValidTransition(from, to JobStatus) bool {
	switch from {
	case StatusQueued:
		return to == StatusRunning || to == StatusCancelled
	case StatusRunning:
		return to == StatusSucceeded || to == StatusFailed || to == StatusCancelled
	case StatusFailed:
		return to == StatusQueued // Allowed for retry
	case StatusSucceeded, StatusCancelled:
		return false // Terminal states
	default:
		return false
	}
}

func (j *Job) TransitionTo(newStatus JobStatus) error {
	if !IsValidTransition(j.Status, newStatus) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, j.Status, newStatus)
	}
	j.Status = newStatus
	j.UpdatedAt = time.Now().UTC()
	return nil
}
