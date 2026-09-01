package storage

import (
	"context"

	"github.com/gotask/gotask/internal/jobs"
)

type Store interface {
	Create(ctx context.Context, job *jobs.Job) error
	GetByID(ctx context.Context, id string) (*jobs.Job, error)
	Update(ctx context.Context, job *jobs.Job) error
	GetNextQueued(ctx context.Context) (*jobs.Job, error)
	RecoverIncompleteJobs(ctx context.Context) error
	Close() error
}
