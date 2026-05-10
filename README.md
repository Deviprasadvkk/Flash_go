# FlashGuard

Distributed High-Concurrency Inventory Engine built in Go.

FlashGuard is a production-inspired backend demo that shows how to safely handle extreme, short-duration spikes in purchase traffic while guaranteeing no oversells, low latency, and durable order persistence.

[GitHub Repository](https://github.com/Deviprasadvkk/Flash_go)

**Executive Summary**
- FlashGuard accepts purchase requests, places them into a resilient queue, and processes them with workers that perform an atomic check-and-decrement on inventory.
- Redis handles the low-latency stock check, Redpanda provides Kafka-compatible buffering, and PostgreSQL stores final orders as the source of truth.

**Key Features**
- Atomic stock control using Redis Lua scripting
- Kafka-compatible queueing with Redpanda for burst smoothing
- PostgreSQL order persistence for durability and auditability
- Go load-test harness for end-to-end validation

**Why This Project Stands Out**
- It demonstrates concurrency control, distributed buffering, and durable persistence in one system.
- It is structured like a real backend service, not just a toy example.
- It includes a repeatable local workflow with Docker Compose and a built-in load test.

**Repository Layout**
- `cmd/server`: main HTTP service and worker wiring
- `cmd/loadtest`: Go load-test harness to simulate high RPS
- `internal/inventory`: inventory interfaces and Redis-backed implementation
- `internal/queue`: in-memory and Kafka/Redpanda queue implementations
- `internal/orders`: Postgres persistence for processed orders
- `docker-compose.yml`: brings up Redis, Redpanda, and Postgres for local testing

**High-Level Architecture (whiteboard description)**
Draw a simple flow with these boxes and arrows:

- Client -> API Gateway / HTTP Service (`cmd/server`) — handles incoming purchase requests and performs lightweight validation.
- HTTP Service -> Queue (Redpanda/Kafka) — publish request as an immutable event. This decouples ingestion from processing and smooths spikes.
- Worker Group -> Redis (atomic Lua script) — each worker atomically checks-and-decrements stock using a single Lua script call; this ensures zero oversell without heavy DB-level locks.
- On successful decrement -> Worker persists order to PostgreSQL (source of truth) and emits a confirmation event.
- Monitoring & Observability: collect metrics (RPS, queue lag, Redis latency, DB write latency) and alerts for backpressure or failures.

When drawing, annotate these important points:
- "Atomicity at cache layer" — Redis Lua script performs CHECK-AND-DECR in one round-trip.
- "Buffering" — Redpanda decouples spikes from DB write rate; workers consume at a steady pace.
- "Durability" — Orders persisted in Postgres ensure recoverability and reporting.

**How to Run**

Prerequisites: Docker & Docker Compose, Go 1.20+.

1. Start services locally:

```bash
docker-compose up -d redis redpanda postgres
```

2. Configure and run the server (example):

```bash
export REDIS_ADDR=localhost:6379
export POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/flashguard
export KAFKA_ADDR=localhost:9092        # optional: enable Redpanda-backed queueing
export KAFKA_TOPIC=flashguard-orders    # optional
go run ./cmd/server
```

3. Submit a test purchase (accepted and queued):

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"item_id":"phone-001","user_id":"user-123"}' \
  http://localhost:8080/purchase
```

4. Run built-in load test to simulate high RPS (example):

```bash
go build ./cmd/loadtest
./loadtest -url http://localhost:8080/purchase -c 100 -rps 5000 -n 20000
```

5. Verify persisted orders in Postgres (psql):

```bash
psql "postgres://postgres:postgres@localhost:5432/flashguard" -c "select * from orders order by created_at desc limit 10;"
```

**Tech Stack Decisions**

- **Go**: chosen for its minimal runtime, small binary footprint, and first-class concurrency (goroutines, channels). Go's performance and simplicity make it ideal for high-throughput backend services where predictable latency and low GC pauses matter.

- **Redis (cache/coordination)**: used for low-latency atomic check-and-decrement operations via an embedded Lua script. Redis provides millisecond-level latency and the Lua scripting environment guarantees atomic execution for the CHECK-AND-DECR pattern without expensive DB locks.

- **Redpanda (Kafka-compatible)**: provides durable, partitioned, and ordered buffering of events. Using a Kafka-compatible broker decouples ingestion from processing, allows horizontal scaling of consumers, and keeps the system resilient under sudden spikes. Redpanda is preferred for local/dev simplicity and Kafka-compatibility in production.

- **PostgreSQL**: chosen as the source-of-truth for orders. Postgres provides strong consistency, ACID guarantees, and works well with optimistic concurrency control (MVCC) for additional data integrity.

**Architecture at a Glance**
- Client sends a purchase request to the Go API.
- The API enqueues the request to Redpanda or the in-memory queue.
- A worker consumes the request and performs an atomic Redis stock decrement.
- If stock is available, the order is written to PostgreSQL.
- The load-test harness can stress the system locally to validate throughput and correctness.

**Production Considerations**
- Use Redis persistence and monitoring to avoid data loss on cache failures. Consider Redis Cluster for scaling and high-availability.
- Configure Redpanda/Kafka with enough partitions to scale consumers; tune retention and acknowledgement levels for your durability vs throughput needs.
- Add idempotency keys to requests and dedup logic on consumers to handle retries safely.
- Monitor: queue lag, consumer lag, Redis error rates, DB write latencies, error rates.
- Security: mutual TLS for services, auth between components (SASL for Kafka, Redis ACLs, DB credentials managed in a secret store).

**Files to Inspect**
- Core HTTP + worker: [cmd/server/main.go](cmd/server/main.go)
- Inventory logic (redis + demo): [internal/inventory/inventory.go](internal/inventory/inventory.go) and [internal/inventory/redis_inventory.go](internal/inventory/redis_inventory.go)
- Queue implementations: [internal/queue/queue.go](internal/queue/queue.go) and [internal/queue/kafka_queue.go](internal/queue/kafka_queue.go)
- Order persistence: [internal/orders/orders.go](internal/orders/orders.go)
- Compose: [docker-compose.yml](docker-compose.yml)

**How I validated the demo**
- I started `redis` and ran the server with `REDIS_ADDR` pointing at the local Redis instance.
- I exercised the `/purchase` endpoint and validated that stock was decremented by the Redis Lua script.
- I added a small Go load-test tool (`cmd/loadtest`) to simulate high RPS and validate the ingress + queue + worker flow locally.

**Next Improvements (optional)**
- Add idempotency keys and dedup store for at-least-once delivery semantics.
- Implement backpressure and circuit-breakers at the API gateway to protect internal systems.
- Add integration tests that spin up the compose stack and run deterministic scenarios.

**Contact / Notes**
If a recruiter or engineer needs a walkthrough, start by reading [cmd/server/main.go](cmd/server/main.go) and then inspect the `internal` packages to see the abstractions and production-ready integrations.

---
This README is written to be recruiter-friendly and highlights design choices and how to run the system locally.
