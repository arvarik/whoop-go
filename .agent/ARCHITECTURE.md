# Architecture

_This document acts as the definitive anchor for understanding system design, data models, API contracts, and technology boundaries. Update this document during the Design and Review phases._

## 1. Tech Stack & Infrastructure
- **Language / Runtime**: Go 1.24.0
- **Frontend**: N/A (Go SDK Library)
- **Backend / API**: N/A (Go SDK Library)
- **Database**: N/A (Go SDK Library)
- **Deployment**: Single binary library imported via `go mod`
- **Package Management**: Go modules (`go.mod`) + Makefile
- **Build System**: `go build` via Makefile

## 2. System Boundaries & Data Flow

### Request / Data Flow
- **Client Configuration**: Initialized using the Functional Options pattern (`whoop.NewClient(whoop.WithToken(...))`).
- **Rate Limiting**: Intrinsic thread-safe token bucket rate-limiter enforcing WHOOP API quotas (100 req/min). Automatically intercepts `429 Too Many Requests`.
- **Retry & Backoff**: Randomized exponential backoff and safe retries on HTTP 429 and transient failures.
- **Pagination**: Transparent iterator pattern utilizing `.NextPage(ctx)` to sequentially traverse all history without manual cursor management.
- **Webhooks**: Inbound HTTP request → memory-capped (1MB) read stream → `TeeReader` → `HMAC-SHA256` authenticity hash validation → returned structured skinny webhook types (`workout.updated`, etc.).

### Concurrency / Threading Model
- **Thread Safety**: The intrinsic token bucket rate limiter is concurrency-safe. The core `Client` can be used safely across multiple goroutines.
- **Webhook Processing**: Designed for high-performance, single-pass webhook stream consumption, suitable for async processing in Go handlers.

## 3. Data Models & Database Schema
- N/A — No database utilized. Exposes strongly-typed Go structs mapping the complex WHOOP domain (Cycles, Workouts, Sleep, Recovery, Profile).

## 4. API Contracts
- The exported public types and functions of the `whoop` package define the API contract.
- Primary interaction point: `*whoop.Client` and its attached sub-services (e.g., `client.Cycle.List(ctx, ...)`).
- Webhook contract: `whoop.ParseWebhook(r *http.Request, secret string) (*whoop.WebhookEvent, error)`.

## 5. External Integrations / AI
- **WHOOP API v2**: The library is exclusively an adapter for the official WHOOP developer API.
- **Dependencies**: Zero external dependencies, utilizing strictly the Go standard library (`net/http`, `crypto/hmac`) and `golang.org/x/time/rate` for the foundational token bucket algorithm.

## 6. Invariants & Safety Rules
- **CRITICAL**: The `.whoop_token.json` file (used in the auth example) contains plaintext OAuth tokens. NEVER commit this to version control.
- **CRITICAL**: Webhook payloads are strictly capped at 1 MB via `io.LimitReader` to prevent memory exhaustion attacks.
- **CRITICAL**: The HTTP handler calling `whoop.ParseWebhook()` MUST NOT consume `r.Body` before invoking the function, as it relies on single-pass stream consumption.
- **CRITICAL**: Backoff durations must never be negative.

## 7. Error Handling Patterns
- Idiomatic Go explicit `if err != nil` propagation.
- Sub-services return typed errors or wrap HTTP transport errors.
- Internal automatic retry mechanism handles transient network failures and specific HTTP status codes (e.g., 429) transparently.

## 8. Directory Structure
- `whoop/` — The core library source code.
- `cmd/example/` — A fully-functional Webhook + REST architecture example application.
- `cmd/auth/` — A helper script to handle the full OAuth 2.0 Authorization Code flow.
- `bin/` — Compiled output directory.

## 9. Local Development
- **Install & Setup**: `make setup` (configures local git hooks).
- **Run Example**: Copy `.env.example` to `.env`, source it, and `make build-local && ./bin/example`.
- **Run Tests**: `make test`
- **Lint**: `make lint`
- **Auth Helper**: `go run cmd/auth/main.go`

## 10. Environment Variables
_(Required for the `cmd/` examples)_

| Variable | Required | Description |
|----------|----------|-------------|
| `WHOOP_CLIENT_ID` | Yes | OAuth 2.0 Client ID |
| `WHOOP_CLIENT_SECRET` | Yes | OAuth 2.0 Client Secret |
| `WHOOP_WEBHOOK_SECRET` | Yes | Secret key to validate incoming webhooks |
| `WHOOP_REDIRECT_URI` | No | Callback URL (default: `http://localhost:8081/callback`) |