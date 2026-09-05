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
	ListByGroupID(ctx context.Context, groupID string) ([]*jobs.Job, error)
	ListGroupIDs(ctx context.Context, limit int) ([]string, error)
	ListRecent(ctx context.Context, limit int) ([]*jobs.Job, error)
	RecoverIncompleteJobs(ctx context.Context) error
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context) (map[jobs.JobStatus]int64, error)
	Close() error
}
