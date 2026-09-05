package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gotask/gotask/internal/jobs"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite db: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		payload TEXT NOT NULL,
		priority INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		attempts INTEGER NOT NULL DEFAULT 0,
		max_attempts INTEGER NOT NULL DEFAULT 3,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		run_at TEXT NOT NULL,
		started_at TEXT,
		completed_at TEXT,
		last_error TEXT,
		group_id TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_status_run_at ON jobs(status, run_at, priority DESC, created_at ASC);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Databases created before group_id existed need the column added.
	if _, err := s.db.Exec(`ALTER TABLE jobs ADD COLUMN group_id TEXT NOT NULL DEFAULT ''`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	_, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_jobs_group_id ON jobs(group_id, created_at ASC)`)
	return err
}

func (s *SQLiteStore) Create(ctx context.Context, j *jobs.Job) error {
	query := `
		INSERT INTO jobs (id, type, payload, priority, status, attempts, max_attempts, created_at, updated_at, run_at, started_at, completed_at, last_error, group_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	payloadStr := string(j.Payload)
	if len(payloadStr) == 0 {
		payloadStr = "{}"
	}

	_, err := s.db.ExecContext(ctx, query,
		j.ID,
		j.Type,
		payloadStr,
		j.Priority,
		string(j.Status),
		j.Attempts,
		j.MaxAttempts,
		j.CreatedAt.UTC().Format(time.RFC3339Nano),
		j.UpdatedAt.UTC().Format(time.RFC3339Nano),
		j.RunAt.UTC().Format(time.RFC3339Nano),
		formatTimePtr(j.StartedAt),
		formatTimePtr(j.CompletedAt),
		j.LastError,
		j.GroupID,
	)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetByID(ctx context.Context, id string) (*jobs.Job, error) {
	query := `
		SELECT id, type, payload, priority, status, attempts, max_attempts, created_at, updated_at, run_at, started_at, completed_at, last_error, group_id
		FROM jobs WHERE id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id)
	return scanJob(row)
}

func (s *SQLiteStore) Update(ctx context.Context, j *jobs.Job) error {
	query := `
		UPDATE jobs
		SET type = ?, payload = ?, priority = ?, status = ?, attempts = ?, max_attempts = ?,
		    updated_at = ?, run_at = ?, started_at = ?, completed_at = ?, last_error = ?
		WHERE id = ?
	`

	payloadStr := string(j.Payload)
	if len(payloadStr) == 0 {
		payloadStr = "{}"
	}

	res, err := s.db.ExecContext(ctx, query,
		j.Type,
		payloadStr,
		j.Priority,
		string(j.Status),
		j.Attempts,
		j.MaxAttempts,
		j.UpdatedAt.UTC().Format(time.RFC3339Nano),
		j.RunAt.UTC().Format(time.RFC3339Nano),
		formatTimePtr(j.StartedAt),
		formatTimePtr(j.CompletedAt),
		j.LastError,
		j.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return jobs.ErrJobNotFound
	}

	return nil
}

func (s *SQLiteStore) GetNextQueued(ctx context.Context) (*jobs.Job, error) {
	query := `
		SELECT id, type, payload, priority, status, attempts, max_attempts, created_at, updated_at, run_at, started_at, completed_at, last_error, group_id
		FROM jobs
		WHERE status = 'queued' AND run_at <= ?
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
	`

	now := time.Now().UTC().Format(time.RFC3339Nano)
	row := s.db.QueryRowContext(ctx, query, now)
	j, err := scanJob(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No jobs available
		}
		return nil, err
	}
	return j, nil
}

func (s *SQLiteStore) RecoverIncompleteJobs(ctx context.Context) error {
	query := `
		UPDATE jobs
		SET status = 'queued', updated_at = ?, last_error = 'recovered from process restart'
		WHERE status = 'running'
	`
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, query, now)
	if err != nil {
		return fmt.Errorf("failed to recover incomplete jobs: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs").Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count jobs: %w", err)
	}
	return count, nil
}

func (s *SQLiteStore) CountByStatus(ctx context.Context) (map[jobs.JobStatus]int64, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT status, COUNT(*) FROM jobs GROUP BY status")
	if err != nil {
		return nil, fmt.Errorf("failed to query job counts: %w", err)
	}
	defer rows.Close()

	result := make(map[jobs.JobStatus]int64)
	for rows.Next() {
		var statusStr string
		var count int64
		if err := rows.Scan(&statusStr, &count); err != nil {
			return nil, fmt.Errorf("failed to scan job count: %w", err)
		}
		result[jobs.JobStatus(statusStr)] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job count iteration error: %w", err)
	}
	return result, nil
}

const jobColumns = `id, type, payload, priority, status, attempts, max_attempts, created_at, updated_at, run_at, started_at, completed_at, last_error, group_id`

func (s *SQLiteStore) queryJobs(ctx context.Context, query string, args ...any) ([]*jobs.Job, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	result := make([]*jobs.Job, 0)
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job iteration error: %w", err)
	}
	return result, nil
}

func (s *SQLiteStore) ListByGroupID(ctx context.Context, groupID string) ([]*jobs.Job, error) {
	return s.queryJobs(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE group_id = ? ORDER BY priority DESC, created_at ASC`, groupID)
}

func (s *SQLiteStore) ListRecent(ctx context.Context, limit int) ([]*jobs.Job, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.queryJobs(ctx,
		`SELECT `+jobColumns+` FROM jobs ORDER BY created_at DESC LIMIT ?`, limit)
}

func (s *SQLiteStore) ListGroupIDs(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT group_id FROM jobs WHERE group_id != '' GROUP BY group_id ORDER BY MIN(created_at) DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query group ids: %w", err)
	}
	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan group id: %w", err)
		}
		result = append(result, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("group id iteration error: %w", err)
	}
	return result, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanJob(s scannable) (*jobs.Job, error) {
	var j jobs.Job
	var payloadStr string
	var statusStr string
	var createdAtStr, updatedAtStr, runAtStr string
	var startedAtStr, completedAtStr sql.NullString

	err := s.Scan(
		&j.ID,
		&j.Type,
		&payloadStr,
		&j.Priority,
		&statusStr,
		&j.Attempts,
		&j.MaxAttempts,
		&createdAtStr,
		&updatedAtStr,
		&runAtStr,
		&startedAtStr,
		&completedAtStr,
		&j.LastError,
		&j.GroupID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, jobs.ErrJobNotFound
		}
		return nil, err
	}

	j.Payload = json.RawMessage(payloadStr)
	j.Status = jobs.JobStatus(statusStr)

	if j.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAtStr); err != nil {
		// fallback to RFC3339
		if j.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr); err != nil {
			return nil, err
		}
	}
	if j.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAtStr); err != nil {
		if j.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr); err != nil {
			return nil, err
		}
	}
	if j.RunAt, err = time.Parse(time.RFC3339Nano, runAtStr); err != nil {
		if j.RunAt, err = time.Parse(time.RFC3339, runAtStr); err != nil {
			return nil, err
		}
	}

	if startedAtStr.Valid {
		if t, err := time.Parse(time.RFC3339Nano, startedAtStr.String); err == nil {
			j.StartedAt = &t
		} else if t, err := time.Parse(time.RFC3339, startedAtStr.String); err == nil {
			j.StartedAt = &t
		}
	}

	if completedAtStr.Valid {
		if t, err := time.Parse(time.RFC3339Nano, completedAtStr.String); err == nil {
			j.CompletedAt = &t
		} else if t, err := time.Parse(time.RFC3339, completedAtStr.String); err == nil {
			j.CompletedAt = &t
		}
	}

	return &j, nil
}

func formatTimePtr(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{
		String: t.UTC().Format(time.RFC3339Nano),
		Valid:  true,
	}
}
