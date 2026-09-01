# GoTask

A production-minded concurrent background job queue built in Go.

## Project Overview

GoTask is a robust, concurrent background job processing system designed to demonstrate production-quality Go backend engineering, clean architecture, priority job queues, worker pools, retry strategies with exponential backoff, persistent state (SQLite), REST API, CLI client, graceful shutdown, and containerization.

## Architecture

- **cmd/server**: HTTP server entrypoint.
- **cmd/gotask**: CLI client entrypoint.
- **internal/api**: REST API router and handlers.
- **internal/config**: Environment-based configuration.
- **internal/logging**: Structured logging (`log/slog`).
- **internal/queue**: Priority job queue.
- **internal/worker**: Concurrent worker pool.
- **internal/jobs**: Job domain logic and handlers.
- **internal/storage**: Persistence layer.

## Getting Started (Phase 1 Foundation)

### Prerequisites

- Go 1.22+ installed.

### Build

```bash
make build
```

### Run Server

```bash
make run
```

### Verify Health

```bash
curl http://localhost:8080/health
```
