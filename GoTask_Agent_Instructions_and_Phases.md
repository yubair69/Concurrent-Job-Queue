# GoTask — Agent Instructions & Implementation Phases

## Project

**GoTask — Concurrent Background Job Queue**

Build a production-quality, portfolio-ready background job processing system in Go. The project should demonstrate practical Go engineering, concurrency, API design, persistence, reliability, testing, observability, and containerization.

The implementation must be completed incrementally through the phases below. Do not skip phases unless the task is genuinely not applicable. Keep the project runnable after every phase.

---

# Agent Instructions

## 1. Primary Objective

Implement GoTask as a clean, maintainable Go backend that:

- Accepts jobs through a REST API.
- Persists job state.
- Processes jobs asynchronously using a concurrent worker pool.
- Supports job priorities.
- Retries failed jobs with exponential backoff.
- Supports cancellation.
- Handles graceful shutdown correctly.
- Exposes useful operational information.
- Provides a CLI client.
- Includes unit and integration tests.
- Runs through Docker.
- Is documented clearly enough for a recruiter or engineer to understand and run.

The final result should be something that can honestly be presented as a serious backend/concurrency project on a resume.

---

## 2. Technical Direction

Use:

- **Go** as the primary language.
- Standard library wherever practical.
- A lightweight HTTP router only if it materially improves maintainability.
- SQLite initially for simplicity; structure the persistence layer so PostgreSQL can be added without rewriting the application.
- Docker / Docker Compose for reproducible execution.
- Go's built-in `testing` package.
- Structured logging using Go's standard `log/slog` where practical.

Do not introduce unnecessary frameworks or dependencies.

---

## 3. Architecture

Keep clear separation between:

```text
cmd/
    server/
    gotask/

internal/
    api/
    queue/
    worker/
    jobs/
    storage/
    config/
    logging/

migrations/

tests/

Dockerfile
docker-compose.yml
go.mod
README.md
```

The exact structure may evolve if a better design is justified, but responsibilities must remain separated.

Recommended flow:

```text
HTTP Client / CLI
       |
       v
   REST API
       |
       v
   Job Service
       |
       v
 Priority Job Queue
       |
       v
 Worker Pool
       |
       v
 Job Handler
       |
       v
 Persistence Layer
```

---

## 4. Engineering Principles

The agent must:

1. Prefer simple, idiomatic Go.
2. Keep functions small and responsibilities focused.
3. Pass `context.Context` through long-running operations.
4. Avoid global mutable state.
5. Make concurrency ownership explicit.
6. Protect shared state correctly.
7. Avoid goroutine leaks.
8. Handle errors explicitly.
9. Add tests alongside implementation.
10. Avoid premature abstractions.
11. Document non-obvious concurrency decisions.
12. Keep public APIs stable once established.
13. Never silently swallow errors.
14. Never use sleeps as synchronization in tests when proper synchronization is possible.
15. Run formatting, tests, and static checks before declaring a phase complete.

---

## 5. Concurrency Requirements

The worker system must demonstrate real concurrency rather than merely spawning goroutines.

Implement:

- Configurable worker count.
- Job dispatching through channels or an equivalent concurrency-safe design.
- Safe job state transitions.
- No duplicate execution caused by race conditions.
- Proper worker lifecycle management.
- Graceful shutdown.
- Context cancellation.
- Protection against goroutine leaks.
- Backpressure or a bounded queue where appropriate.

Use:

```bash
go test -race ./...
```

as a mandatory validation command once concurrency exists.

---

## 6. Job Model

A job should contain at minimum:

- ID
- Type
- Payload
- Priority
- Status
- Attempt count
- Maximum attempts
- Created timestamp
- Updated timestamp
- Scheduled/available timestamp
- Started timestamp
- Completed timestamp
- Error information where applicable

Suggested states:

```text
queued
running
succeeded
failed
cancelled
```

Define valid state transitions and enforce them.

---

## 7. Queue Requirements

The queue must support:

- FIFO behavior within the same priority.
- Multiple priority levels.
- Concurrent producers.
- Concurrent consumers.
- Bounded capacity.
- Safe enqueue/dequeue behavior.
- Shutdown behavior.

Do not implement priorities by repeatedly sorting a large slice on every operation if a more appropriate data structure is practical.

Document the chosen queue design and its trade-offs.

---

## 8. Retry & Backoff

Failed jobs should be retried when configured to do so.

Implement exponential backoff similar to:

```text
delay = baseDelay * 2^attempt
```

Include:

- Configurable base delay.
- Maximum retry delay.
- Maximum attempts.
- Retry only for retryable failures.
- Persistence of attempt count.
- Re-queueing after the delay.

Avoid uncontrolled retry storms.

Add tests for:

- First retry.
- Multiple retries.
- Maximum attempts.
- Backoff calculation.
- Permanent failure.
- Cancellation during retry delay.

---

## 9. REST API

Implement:

### Create Job

```http
POST /jobs
```

Accept:

- Type
- Payload
- Priority
- Maximum attempts

Return:

- Job ID
- Initial status
- Relevant metadata

### Get Job

```http
GET /jobs/:id
```

Return current state and useful metadata.

### Cancel Job

```http
DELETE /jobs/:id
```

Cancellation must have well-defined semantics for queued and running jobs.

### Operational Endpoint

Implement at least:

```http
GET /health
```

and preferably:

```http
GET /metrics
```

or another simple operational statistics endpoint.

Define appropriate HTTP status codes and JSON error responses.

---

## 10. Job Handlers

Do not hard-code all processing into the worker.

Create a handler abstraction such as:

```text
Job Type -> Handler
```

Include at least a few demonstration handlers, for example:

- `echo`
- `sleep`
- `email` simulation

The handlers should make it easy to add new job types without modifying worker internals.

---

## 11. Persistence

Persist job state so jobs survive process restarts.

At minimum persist:

- Job metadata.
- Status.
- Attempts.
- Retry scheduling information.
- Error information.
- Timestamps.

On startup:

- Detect jobs left in an in-progress state.
- Apply a clearly documented recovery policy.
- Requeue recoverable jobs where appropriate.

Keep storage behind an interface so SQLite and PostgreSQL implementations can be swapped later.

---

## 12. CLI

Create:

```bash
gotask submit --type=email --payload=data.json
gotask status <job-id>
gotask cancel <job-id>
gotask workers
```

The CLI should communicate with the REST API rather than directly accessing the database.

Provide useful human-readable output and appropriate non-zero exit codes on failure.

---

## 13. Logging & Observability

Use structured logs.

Include useful fields such as:

- job ID
- job type
- worker ID
- attempt
- status
- duration
- error

Do not log sensitive job payloads by default.

Add basic operational statistics such as:

- queued jobs
- running jobs
- succeeded jobs
- failed jobs
- cancelled jobs
- active workers
- total processed jobs

---

## 14. Graceful Shutdown

The server must handle OS termination signals.

Shutdown sequence should be deliberate:

```text
Receive SIGINT/SIGTERM
        ↓
Stop accepting new work
        ↓
Stop queue intake
        ↓
Allow active jobs to finish up to timeout
        ↓
Cancel remaining work
        ↓
Persist final state
        ↓
Close database/resources
        ↓
Exit
```

Use configurable shutdown timeouts.

Test shutdown behavior.

---

## 15. Testing Requirements

Testing is a core deliverable, not an afterthought.

Include:

### Unit tests

Test:

- Queue operations.
- Priority ordering.
- Job state transitions.
- Retry logic.
- Backoff calculation.
- Handler registry.
- Storage operations.
- API validation.

### Concurrency tests

Test:

- Multiple producers.
- Multiple workers.
- Queue capacity.
- Cancellation.
- Shutdown.
- Duplicate execution prevention.

Run:

```bash
go test -race ./...
```

### Integration tests

Cover flows such as:

```text
Create job
   ↓
Persist job
   ↓
Worker executes job
   ↓
Status becomes succeeded
```

and:

```text
Create failing job
   ↓
Retry
   ↓
Retry
   ↓
Final failure
```

Also test restart/recovery behavior.

---

## 16. Docker

Provide:

- `Dockerfile`
- `docker-compose.yml` where useful
- Environment-based configuration
- Persistent storage for the database

The project must be runnable with a small number of commands.

Document:

```bash
docker compose up --build
```

and local Go execution.

---

## 17. Configuration

Support configuration through environment variables and/or flags.

At minimum:

- HTTP port
- Database path/DSN
- Worker count
- Queue capacity
- Default retry count
- Retry base delay
- Maximum retry delay
- Shutdown timeout
- Log level

Provide sensible defaults.

---

## 18. Documentation

README.md must include:

1. Project overview.
2. Architecture diagram.
3. Features.
4. Why the project exists.
5. Tech stack.
6. Directory structure.
7. Local setup.
8. Docker setup.
9. API examples.
10. CLI examples.
11. Configuration.
12. Concurrency design.
13. Retry/backoff design.
14. Persistence/recovery behavior.
15. Testing commands.
16. Design trade-offs.
17. Future improvements.

Include example `curl` commands.

---

# Phase Plan

## Phase 1 — Project Foundation

### Goals

Create the Go project and establish a clean baseline.

### Tasks

- Initialize Go module.
- Create initial directory structure.
- Add configuration package.
- Add structured logging.
- Create server entrypoint.
- Create basic HTTP server.
- Add `/health`.
- Add graceful HTTP server lifecycle skeleton.
- Add README foundation.
- Add `.gitignore`.
- Add basic Makefile or task runner commands.

### Exit Criteria

```bash
go build ./...
go test ./...
go vet ./...
```

all pass.

---

# Phase 2 — Job Domain & Persistence

### Goals

Define the job model and persistence layer.

### Tasks

- Implement Job model.
- Define job statuses.
- Define valid state transitions.
- Define storage interface.
- Implement SQLite storage.
- Add schema/migrations.
- Implement create/get/update operations.
- Add timestamps and attempt tracking.
- Add storage unit tests.

### Exit Criteria

Jobs can be created, persisted, retrieved, updated, and survive process restarts.

---

# Phase 3 — Concurrent Priority Queue

### Goals

Build the core concurrent queue.

### Tasks

- Implement queue abstraction.
- Implement priority ordering.
- Implement FIFO ordering within priority.
- Add bounded capacity.
- Add concurrent enqueue/dequeue.
- Add shutdown semantics.
- Add context cancellation.
- Add concurrency tests.
- Run race detector.

### Exit Criteria

```bash
go test -race ./...
```

passes with no race conditions.

The queue must correctly handle concurrent producers and consumers.

---

# Phase 4 — Worker Pool & Job Handlers

### Goals

Build asynchronous execution.

### Tasks

- Implement configurable worker pool.
- Implement worker lifecycle.
- Implement handler interface.
- Implement handler registry.
- Add sample handlers.
- Connect queue to workers.
- Persist running/succeeded/failed states.
- Record execution duration.
- Prevent duplicate execution.

### Exit Criteria

Submitting a job results in asynchronous processing by one of the workers.

Multiple jobs execute concurrently when multiple workers are configured.

---

# Phase 5 — Retry & Exponential Backoff

### Goals

Add reliable failure handling.

### Tasks

- Implement retry policy.
- Implement exponential backoff.
- Add maximum retry delay.
- Add maximum attempts.
- Distinguish retryable and permanent failures.
- Persist retry metadata.
- Requeue retryable jobs.
- Handle cancellation during backoff.
- Add comprehensive tests.

### Exit Criteria

A failing retryable job follows:

```text
queued
→ running
→ failed
→ queued
→ running
→ ...
→ succeeded / failed
```

according to the configured retry policy.

---

# Phase 6 — REST API

### Goals

Expose the queue through a usable HTTP API.

### Tasks

Implement:

```http
POST   /jobs
GET    /jobs/:id
DELETE /jobs/:id
GET    /health
GET    /metrics
```

Also:

- Request validation.
- JSON serialization.
- HTTP status codes.
- Error responses.
- API tests.
- Context propagation.
- Request logging.

### Exit Criteria

The complete job lifecycle can be controlled using HTTP requests.

---

# Phase 7 — Cancellation & Graceful Shutdown

### Goals

Make lifecycle behavior production-quality.

### Tasks

- Implement queued-job cancellation.
- Implement running-job cancellation through context.
- Add SIGINT/SIGTERM handling.
- Stop accepting new work during shutdown.
- Drain active jobs within timeout.
- Cancel remaining jobs.
- Persist final states.
- Recover interrupted jobs on restart.
- Add shutdown/recovery tests.

### Exit Criteria

The application can be terminated and restarted without corrupting job state or leaking goroutines.

---

# Phase 8 — CLI

### Goals

Provide a developer-friendly command-line client.

### Tasks

Implement:

```bash
gotask submit
gotask status
gotask cancel
gotask workers
```

Add:

- Flags.
- JSON payload support.
- Human-readable output.
- API error handling.
- Exit codes.
- CLI tests.

### Exit Criteria

A user can operate the system without manually writing HTTP requests.

---

# Phase 9 — Observability & Production Hardening

### Goals

Make the project feel production-ready.

### Tasks

- Structured logging throughout.
- Worker IDs.
- Job execution metrics/statistics.
- Queue depth statistics.
- Request logging.
- Configurable log level.
- Timeouts.
- Defensive input limits.
- Better error classification.
- Review resource cleanup.
- Review goroutine lifecycle.
- Review race safety.

### Exit Criteria

The system provides enough operational information to understand what it is doing during execution.

---

# Phase 10 — Docker & Deployment

### Goals

Make the project reproducible.

### Tasks

- Create multi-stage Dockerfile if appropriate.
- Add docker-compose configuration.
- Configure environment variables.
- Persist database data.
- Add container healthcheck.
- Verify container startup.
- Verify API access.
- Verify worker processing inside container.
- Document Docker workflow.

### Exit Criteria

This works:

```bash
docker compose up --build
```

and the complete application can process jobs.

---

# Phase 11 — Comprehensive Testing & Quality Gate

### Goals

Perform a full engineering-quality pass.

### Tasks

Run:

```bash
go test ./...
go test -race ./...
go vet ./...
go fmt ./...
```

Where available, also run static analysis/linting.

Verify:

- Unit tests.
- Integration tests.
- API tests.
- Queue concurrency.
- Retry behavior.
- Cancellation.
- Shutdown.
- Restart recovery.
- Docker execution.

Fix every race, flaky test, build error, and obvious reliability issue found.

### Exit Criteria

All automated tests and quality checks pass consistently.

---

# Phase 12 — Documentation, Resume Readiness & Final Review

### Goals

Turn the implementation into a polished portfolio project.

### Tasks

- Finish README.
- Add architecture diagram.
- Add API examples.
- Add CLI examples.
- Document concurrency model.
- Document retry/backoff.
- Document recovery behavior.
- Document design trade-offs.
- Add sample configuration.
- Add demo workflow.
- Review naming and package structure.
- Remove dead code.
- Remove unnecessary dependencies.
- Improve comments where useful.
- Verify clean clone/build experience.

Create a final project summary containing:

- What was built.
- Key technical challenges.
- Concurrency design.
- Reliability mechanisms.
- Testing strategy.
- Deployment approach.
- Potential future improvements.

### Exit Criteria

A new developer can clone the repository, run it, submit jobs, observe workers processing them, run tests, and understand the architecture from the README.

---

# Definition of Done

The project is complete only when all of the following are true:

- [ ] Go project builds successfully.
- [ ] REST API works.
- [ ] Jobs persist to SQLite.
- [ ] Multiple workers process jobs concurrently.
- [ ] Priority ordering works.
- [ ] Retry logic works.
- [ ] Exponential backoff works.
- [ ] Job cancellation works.
- [ ] Graceful shutdown works.
- [ ] Restart/recovery behavior is implemented.
- [ ] CLI works.
- [ ] Structured logging exists.
- [ ] Operational statistics exist.
- [ ] Docker deployment works.
- [ ] Unit tests exist.
- [ ] Integration tests exist.
- [ ] Race detector passes.
- [ ] `go vet` passes.
- [ ] Documentation is complete.
- [ ] No obvious goroutine leaks remain.
- [ ] No known race conditions remain.
- [ ] No unnecessary dependencies remain.
- [ ] Project can be demonstrated end-to-end.

---

# Agent Working Protocol

For every phase:

1. Inspect the existing implementation before changing it.
2. Understand current architecture and constraints.
3. Implement the smallest coherent increment.
4. Add/update tests immediately.
5. Run formatting.
6. Run relevant tests.
7. Run the race detector whenever concurrency-related code changes.
8. Fix failures before moving on.
9. Update documentation when behavior changes.
10. Summarize what changed and what remains.
11. Do not proceed to the next phase if the current phase's exit criteria are not satisfied.

When making architectural decisions, favor correctness and clarity over cleverness.

Do not add features merely to make the project larger. Every feature should reinforce the project's core objective: demonstrating strong Go backend and concurrency engineering.

---

# Suggested Final Resume Bullet

> Built a concurrent background job processing system in Go using goroutines, worker pools, priority queues, persistent job state, retry/exponential-backoff strategies, graceful shutdown, and a REST/CLI interface; containerized the system with Docker and validated concurrency with integration tests and Go's race detector.

# Suggested Repository Tagline

> A production-minded concurrent background job queue built in Go.

# Recommended Phase Count

**12 phases**

Estimated scope: substantial portfolio project. Prioritize correctness, testing, and clean engineering over rushing through all features.
