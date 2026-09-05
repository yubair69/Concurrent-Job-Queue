# PixelForge

A concurrent media processing platform powered by the GoTask engine.

Upload one image or video and PixelForge expands it into a set of independent
processing jobs — metadata, thumbnail, resizes, compression, transcodes — which
are handed to GoTask's priority queue and executed concurrently by a pool of Go
workers. The web UI shows the real queue depth, real per-worker activity and
real results as they land; nothing on screen is simulated.

```text
UPLOAD → PixelForge API → GoTask priority queue → worker pool → outputs
```

## PixelForge

### Run it — one command

```powershell
.\run.ps1          # Windows / PowerShell
```

```bash
./run.sh           # macOS / Linux / Git Bash
```

That installs frontend dependencies if needed, builds the UI when sources have
changed, and starts the server. Open **http://localhost:8080**, drop in a file
and press Process.

Options: `.\run.ps1 -Port 9000 -Workers 8` (PowerShell) or
`GOTASK_PORT=9000 GOTASK_WORKER_COUNT=8 ./run.sh`; add `-Fresh` / `--fresh` to
force a frontend rebuild.

<details>
<summary>Manual equivalent</summary>

```bash
cd web && npm install && npm run build && cd ..
go run ./cmd/server          # serves the API and the built UI on :8080
```

For frontend development, `npm run dev` proxies `/api` to the Go server on :8080.
</details>

### Jobs created per upload

| Media | Jobs |
|-------|------|
| Image | metadata, thumbnail, resize 1280px, compress, optimized version |
| Video | metadata, thumbnail, extract audio, transcode 480p/720p/1080p |

Cheap jobs are queued at a higher priority than heavy ones, so quick results
surface first while transcodes are still running — a direct use of GoTask's
priority queue.

### PixelForge API

| Endpoint | Purpose |
|----------|---------|
| `POST /api/uploads` | multipart upload; creates one job per pipeline step |
| `GET /api/uploads` | recent uploads |
| `GET /api/uploads/:id` | aggregate status of one upload's jobs |
| `GET /api/jobs` | recent job history |
| `GET /api/workers` | live per-worker state |
| `GET /api/outputs/:id/:file` | processed output files |
| `GET /api/health`, `GET /api/metrics` | aliases of the core GoTask endpoints |

### Media processing engines

- **Images are processed for real** with pure Go (`image` + `golang.org/x/image/draw`) —
  no external binaries, works out of the box.
- **Video uses ffmpeg** (`ffmpeg`/`ffprobe` on `PATH`) for genuine transcoding,
  thumbnails and audio extraction. If ffmpeg is absent, video jobs still run
  through the real queue, workers and retry logic but their outputs are
  **simulated**, and both the API (`"engine":"simulated"`) and the UI say so
  rather than presenting a fake result as real. Install ffmpeg and restart for
  real transcoding.
- The "optimized version" step emits an optimized JPEG rather than WebP: there
  is no pure-Go WebP encoder without cgo, so the label reflects what actually
  happens.

### Configuration (PixelForge)

| Environment Variable | Default | Description |
|----------------------|---------|-------------|
| `PIXELFORGE_UPLOAD_DIR` | `data/uploads` | Where uploaded originals are stored |
| `PIXELFORGE_OUTPUT_DIR` | `data/outputs` | Where processed outputs are written |
| `PIXELFORGE_STATIC_DIR` | `web/dist` | Built frontend to serve (unset to disable) |
| `PIXELFORGE_MAX_UPLOAD_MB` | 200 | Maximum upload size |

---

# GoTask — the engine underneath

A production-minded concurrent background job queue built in Go. PixelForge is
a product layer on top of it: media jobs are ordinary GoTask jobs, running
through the same queue, worker pool, retry/backoff, cancellation and SQLite
persistence as any other job type. The original REST API and CLI are unchanged.

## Project Overview

GoTask is a robust, concurrent background job processing system designed to demonstrate production-quality Go backend engineering, clean architecture, priority job queues, worker pools, retry strategies with exponential backoff, persistent state (SQLite), REST API, CLI client, graceful shutdown, and containerization.

## Architecture

```text
HTTP Client / CLI           REST API
       |                        |
       v                        v
    HTTP Request  →  POST /jobs, GET /jobs/:id, DELETE /jobs/:id, GET /health, GET /metrics
                          |
                          v
                       Job Service
                    (internal/api)
                          |
                          v
                   Priority Job Queue
                    (internal/queue)
                          |
                          v
                    Worker Pool
                    (internal/worker)
                          |
                          v
                   Job Handlers
                    (internal/jobs)
                          |
                          v
                   Persistence Layer
                    (internal/storage)
```

- **cmd/server**: HTTP server entrypoint.
- **cmd/gotask**: CLI client entrypoint.
- **internal/api**: REST API router and handlers.
- **internal/cli**: CLI command logic.
- **internal/config**: Environment-based configuration.
- **internal/logging**: Structured logging (`log/slog`).
- **internal/queue**: Concurrent priority job queue.
- **internal/worker**: Concurrent worker pool.
- **internal/jobs**: Job domain model, statuses, transitions, retry logic, and handlers.
- **internal/storage**: Persistence layer (SQLite).
- **migrations**: Database schema.

## Features

- REST API for job submission, status, cancellation, health, and metrics
- Concurrent worker pool with configurable worker count
- Priority job queue with FIFO ordering within priority levels
- Exponential backoff retry strategy with configurable base/max delays
- Automatic job recovery on restart
- Graceful shutdown with configurable timeout
- Structured JSON logging
- SQLite persistence with WAL mode
- Docker ready with multi-stage builds
- Developer-friendly CLI client
- Comprehensive unit and integration tests

## Tech Stack

- **Language**: Go (1.23+)
- **HTTP**: Standard library `net/http`
- **Storage**: SQLite (pure-Go driver via `modernc.org/sqlite`)
- **Logging**: `log/slog` with JSON handler
- **Containerization**: Docker multi-stage build

## Why the Project Exists

GoTask was built as a portfolio project to demonstrate production-quality Go backend engineering. It showcases concurrency patterns (worker pools, channels, contexts), clean architecture with separation of concerns, persistence with recovery semantics, retry strategies, graceful shutdown, observability, and containerization. It serves as a reference implementation that can be discussed in technical interviews.

## Demo Workflow

### Start the Server

```bash
# Local development
go run ./cmd/server

# Or with Docker
docker compose up --build
```

### Submit and Track a Job

```bash
# 1. Submit a job
ID=$(curl -s -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"type":"echo","payload":{"msg":"hello"}}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "Submitted job: $ID"

# 2. Check status
curl http://localhost:8080/jobs/$ID

# 3. Check metrics
curl http://localhost:8080/metrics

# 4. Using CLI (same operations)
gotask submit --type=echo --payload='{"msg":"hello"}'
gotask status $ID
gotask workers
```

### Graceful Shutdown

```bash
# Send SIGINT (Ctrl+C) or SIGTERM — workers finish current jobs before exiting
kill -TERM $(pgrep gotask-server)
```

## Getting Started

### Prerequisites

- Go 1.23+ installed (for local development)
- Docker and Docker Compose (for containerized deployment)

### Local Build

```bash
make build
```

### Run Server Locally

```bash
make run
```

### Verify Health

```bash
curl http://localhost:8080/health
```

### Run Tests

```bash
make test
make race
```

## Docker Setup

### Build and Run with Docker Compose

```bash
docker compose up --build
```

The server will start on port 8080 with a persistent SQLite volume.

### Stop the Server

```bash
docker compose down
```

### Run with Docker Directly

```bash
docker build -t gotask .
docker run -p 8080:8080 -v gotask-data:/var/lib/gotask gotask
```

### Docker Configuration

Environment variables:

| Environment Variable | Default | Description |
|----------------------|---------|-------------|
| `GOTASK_PORT` | 8080 | HTTP server port |
| `GOTASK_DB_PATH` | `/var/lib/gotask/gotask.db` | SQLite database path |
| `GOTASK_WORKER_COUNT` | 4 | Number of worker goroutines |
| `GOTASK_QUEUE_CAPACITY` | 1000 | Maximum queued jobs |
| `GOTASK_DEFAULT_RETRIES` | 3 | Default retry attempts |
| `GOTASK_RETRY_BASE_DELAY` | 1s | Base backoff delay |
| `GOTASK_RETRY_MAX_DELAY` | 30s | Maximum backoff delay |
| `GOTASK_SHUTDOWN_TIMEOUT` | 10s | Graceful shutdown timeout |
| `GOTASK_LOG_LEVEL` | info | Log level (debug, info, warn, error) |

## API Examples

### Create a Job

```bash
# Using curl
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"type":"echo","payload":{"message":"hello"},"priority":5,"max_attempts":3}'

# Using CLI
gotask submit --type=echo --payload='{"message":"hello"}' --priority=5 --retries=3
gotask submit --type=email --payload=@email_data.json
```

Response (`HTTP 201 Created`):

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "type": "echo",
  "priority": 5,
  "max_attempts": 3
}
```

### Get Job Status

```bash
# Using curl
curl http://localhost:8080/jobs/<job-id>

# Using CLI
gotask status <job-id>
```

### Cancel a Job

```bash
# Using curl
curl -X DELETE http://localhost:8080/jobs/<job-id>

# Using CLI
gotask cancel <job-id>
```

### Health Check

```bash
curl http://localhost:8080/health
```

Response (`HTTP 200 OK`):

```json
{"status":"healthy","timestamp":"2024-01-01T12:00:00Z"}
```

### Metrics

```bash
curl http://localhost:8080/metrics
```

Response (`HTTP 200 OK`):

```json
{
  "queued": 10,
  "running": 2,
  "succeeded": 45,
  "failed": 3,
  "cancelled": 1,
  "active_workers": 2,
  "total_workers": 4,
  "total_processed": 51,
  "total_attempted": 55,
  "queue_depth": 10,
  "queue_enqueued": 56,
  "queue_dequeued": 46,
  "queue_capacity": 1000
}
```

## CLI Usage

```bash
# Submit a job with inline JSON payload
gotask submit --type=echo --payload='{"message":"hello world"}'

# Submit a job with payload from file
gotask submit --type=email --payload=@email_data.json --priority=5 --retries=3

# Get job status
gotask status <job-id>

# Cancel a job
gotask cancel <job-id>

# View system metrics
gotask workers
```

## Concurrency Design

The queue uses Go's `container/heap` package for efficient priority-based ordering. Each job is assigned an incrementing sequence number for FIFO ordering within the same priority. Thread safety is provided by `sync.Mutex` and `sync.Cond` for blocking enqueue/dequeue operations. The worker pool spawns configurable goroutines that dequeue and execute jobs. Each job receives its own cancellable context (derived from the worker pool's context) to support individual job cancellation and graceful shutdown. See `internal/queue/queue.go` and `internal/worker/pool.go` for details.

## Retry/Backoff Design

Failed jobs with retryable errors are retried using exponential backoff: `delay = baseDelay * 2^(attempt-1)`. The delay is capped at `maxDelay`. Retryable jobs are transitioned to `queued` status with a future `runAt` timestamp and re-enqueued into the queue after the backoff delay. Permanent failures (`jobs.NewPermanentError`) are not retried. When the maximum number of attempts is reached, the job is marked as `failed` permanently.

## Persistence/Recovery Behavior

Jobs are persisted to SQLite with WAL mode for durability. On startup, any jobs left in the `running` state (due to an unclean shutdown) are automatically recovered to the `queued` state, ready to be reprocessed.

## Testing Commands

```bash
# Run all tests
make test

# Run tests with race detector
make race

# Run vet
make vet

# Format code
make fmt
```

## Design Trade-offs

- **SQLite over PostgreSQL**: Chosen for simplicity and to avoid external service dependencies. The storage layer is behind an interface for easy swap.
- **Pure Go SQLite**: `modernc.org/sqlite` avoids CGO dependencies and simplifies containerization.
- **Single binary**: Server is a single Go binary for easy deployment.
- **No job result storage**: Job payloads and results are not stored permanently (only job metadata). This could be added if needed.
- **In-memory retry scheduling**: Uses `time.AfterFunc` for re-enqueueing; if the worker pool is shut down before the timer fires, the retry is silently dropped. A more robust approach would persist schedules in the database and poll for due jobs.

## Future Improvements

- Add PostgreSQL storage implementation
- Add job result storage and TTL for cleanup
- Add authentication and authorization for the REST API
- Add Prometheus-compatible metrics endpoint
- Add a web dashboard for job monitoring
- Support delayed jobs with cron-like scheduling
- Add job batching and chaining support
