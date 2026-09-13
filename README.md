# load-dancer

A Layer 7 load balancer and reverse proxy written in Go, built as a staged learning project toward a small backend/infrastructure platform.

## What it does

`load-dancer` sits in front of a set of backend HTTP servers and distributes incoming requests using round-robin selection, while handling backend failures through health checks, retries, and circuit breaking.

* **Round-robin routing** across a configurable set of backends
* **Active health checking** - each backend is probed independently on its `/health` endpoint at a fixed interval and marked unhealthy after 3 consecutive failures
* **Per-backend circuit breaking** - a backend that repeatedly fails is temporarily taken out of rotation (`Open`), then given a single trial request after a cooldown (`HalfOpen`) before being trusted again (`Closed`)
* **Automatic retry with backoff** - failed requests are retried against a different healthy backend using exponential backoff with jitter, and only for idempotent HTTP methods (`GET`, `HEAD`, `OPTIONS`, `TRACE`). Non-idempotent methods such as `POST` are never silently retried
* **Graceful shutdown** - on `SIGINT`, `SIGTERM`, the load balancer stops accepting new connections, allows in-flight requests to finish within a bounded timeout, and cleanly stops health-check goroutines
* **Structured logging** - requests are logged with method, path, status code, latency, and a request ID that is propagated through the request lifecycle

## Architecture

```text
                    ┌─────────────────┐
   client ────────▶ │  Load Balancer  │
                    │   port :8080    │
                    └────────┬────────┘
                             │
                 round-robin + health-aware
                       backend selection
                             │
        ┌────────────────────┼────────────────────┐
        ▼                    ▼                    ▼
 ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
 │  backend 1  │      │  backend 2  │      │  backend 3  │
 │    :8001    │      │    :8002    │      │    :8003    │
 └─────────────┘      └─────────────┘      └─────────────┘
        ▲                    ▲                    ▲
        └──────── health checks every 1s ───────┘
```

The load balancer maintains backend state including health and circuit-breaker state. Backend selection is performed for each request attempt, allowing retries to select a different healthy backend.

`cmd/mockbackend` provides a minimal HTTP server for local development and testing. It is a test fixture rather than part of the load balancer itself.

## Design decisions

### Health checks and circuit breakers are separate

Health checking and circuit breaking provide complementary signals:

* **Health checking** is proactive and independent of live traffic.
* **Circuit breaking** reacts to failures observed during actual requests.

A backend can therefore be considered healthy by the health checker while its circuit remains open because it is failing real requests.

### Proxy failures do not directly determine health

Backend health is determined by the independent periodic health check rather than individual proxy failures.

The circuit breaker separately reacts to retryable failures observed during live traffic.

### Retries are limited to idempotent methods

Retries are currently limited to:

```text
GET
HEAD
OPTIONS
TRACE
```

Blindly retrying a non-idempotent request such as `POST` can cause duplicate side effects if the original request reached the backend successfully but its response was lost.

Selected non-idempotent operations could later support retries through mechanisms such as idempotency keys.

## Running locally

Start three mock backends:

```bash
make start-backends
```

Run the load balancer:

```bash
go run ./cmd/loadbalancer
```

Send requests:

```bash
curl http://localhost:8080
```

Stop the mock backends:

```bash
make stop-backends
```

A mock backend can also simulate slowness for testing timeout and retry behavior:

```bash
go run ./cmd/mockbackend -delay http://localhost:8001
```

## Testing

```bash
go test -race ./...
```

Tests currently cover:

* round-robin distribution
* concurrent requests
* health-check state transitions
* retry behavior
* retry backoff and method safety
* circuit-breaker state transitions
* graceful shutdown

## Roadmap

The project is being built incrementally.

### Completed

* Round-robin routing
* Active health checks
* Automatic retries
* Exponential backoff with jitter
* Per-backend circuit breakers
* Graceful shutdown
* Structured logging
* Concurrent request handling and race testing

### Next

* **Control API** for dynamically registering, updating, and removing backends
* **Real Go backend application** with PostgreSQL
* `pgx` + `sqlc` + database migrations
* Authentication and authorization
* Rate limiting

### Later

* Caching
* Other Load Balancing algorithms
* Load testing and performance analysis
* Docker / Docker Compose
* Metrics and distributed tracing
* Kubernetes
* Further distributed-systems features
