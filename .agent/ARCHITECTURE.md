# Architecture

_This document acts as the definitive anchor for understanding system design, data models, API contracts, and technology boundaries. Update this document during the Design and Review phases._

## 1. Tech Stack & Infrastructure
- **Language / Runtime**: Go 1.24.0+
- **Frontend / UI**: None (Go SDK Library)
- **Backend / Database**: None
- **Deployment**: Single library archive imported via standard Go modules (`go get github.com/arvarik/whoop-go`)
- **Package Management**: Go modules (`go.mod`) + Makefile for local automation
- **Build System**: Standard `go build`

## 2. Directory Structure & Internal Package Mapping

### Core Library (`whoop/`)
The flat `whoop/` package is the single importable unit. All source files live at the top level—no sub-packages.

| File | Role |
|------|------|
| `client.go` | Core `Client` struct, `Do()` method (authentication, rate limiting, retry loop), `Get()` convenience helper. Implements `fmt.Stringer` and `fmt.GoStringer` to redact tokens in logs. |
| `options.go` | Functional Options pattern: `WithToken()`, `WithBaseURL()`, `WithHTTPClient()`, `WithMaxRetries()`, `WithBackoffBase()`, `WithBackoffMax()`, `WithRateLimiting()`. |
| `ratelimit.go` | Thread-safe token bucket rate limiter (`golang.org/x/time/rate`) configured for 100 req/min with burst of 100. Uses `atomic.Bool` for toggling. Also contains `calculateBackoff()` with full jitter. |
| `pagination.go` | `ListOptions` struct (Limit, Start, End, NextToken), URL query encoder, and generic `paginatedResponse[T]` type using Go generics. |
| `webhooks.go` | `ParseWebhook()`: memory-capped `io.LimitReader` (1MB) → `io.TeeReader` → `crypto/hmac` SHA-256 → `base64.StdEncoding` signature comparison. Returns `*WebhookEvent` (skinny payload). |
| `errors.go` | Three typed errors: `APIError` (generic HTTP errors), `RateLimitError` (429 with `RetryAfter`), `AuthError` (401/403). All implement `Unwrap()` for `errors.Is()`/`errors.As()`. `mapHTTPError()` dispatches by status code. |
| `scopes.go` | OAuth 2.0 scope constants (`ScopeOffline`, `ScopeReadRecovery`, `ScopeReadCycles`, etc.) as the `Scope` type. |
| `doc.go` | Package-level godoc with Quick Start, Pagination, and Webhook examples. |

### Domain Services & Types
Each domain maps 1:1 to a WHOOP API resource:

| File | Service | Key Types | Methods |
|------|---------|-----------|---------|
| `cycle.go` | `CycleService` | `Cycle`, `Score`, `CyclePage` | `GetByID(ctx, int)`, `List(ctx, *ListOptions)`, `CyclePage.NextPage(ctx)` |
| `workout.go` | `WorkoutService` | `Workout`, `WorkoutScore`, `ZoneDurations`, `WorkoutPage` | `GetByID(ctx, string)`, `List(ctx, *ListOptions)`, `WorkoutPage.NextPage(ctx)` |
| `sleep.go` | `SleepService` | `Sleep`, `SleepScore`, `StageSummary`, `SleepNeeded`, `SleepPage` | `GetByID(ctx, string)`, `List(ctx, *ListOptions)`, `SleepPage.NextPage(ctx)` |
| `recovery.go` | `RecoveryService` | `Recovery`, `RecoveryScore`, `RecoveryPage` | `GetByID(ctx, int)`, `List(ctx, *ListOptions)`, `RecoveryPage.NextPage(ctx)` |
| `profile.go` | `UserService` | `BasicProfile`, `BodyMeasurement` | `GetBasicProfile(ctx)`, `GetBodyMeasurement(ctx)` |

### Executables (`cmd/`)
- **`cmd/example/`**: Reference webhook listener app. Demonstrates `ParseWebhook()`, worker pool pattern (5 goroutines, buffered channel of 100), and REST API pulls triggered by skinny webhook events. Uses `WHOOP_OAUTH_TOKEN` and `WHOOP_WEBHOOK_SECRET` env vars.
- **`cmd/auth/`**: Standalone OAuth 2.0 Authorization Code flow helper. Starts a local HTTP server, handles the browser callback, exchanges the auth code for tokens, saves to `.whoop_token.json`, and supports automatic token refresh. Uses `WHOOP_CLIENT_ID` and `WHOOP_CLIENT_SECRET` env vars.

### Supporting Directories
- **`docs/`**: Contains `archive/`, `designs/`, `explorations/`, `plans/` subdirectories for design documents.
- **`.githooks/`**: Contains a `pre-commit` hook that runs `make lint` before every commit.
- **`.github/workflows/`**: Contains `ci.yml` — GitHub Actions CI pipeline (tidy, vet, test with race/cover, golangci-lint).
- **`bin/`**: Compiled output directory (gitignored).

## 3. System Boundaries & Data Flow

### Request Lifecycle
1. **Bootstrapping**: Consumer invokes `whoop.NewClient(whoop.WithToken("..."))`. Options configure the internal `http.Client` (default 30s timeout), backoff (base: 1s, max: 60s), retries (default: 3), and rate limiter.
2. **Request Execution**: Service methods (e.g., `client.Workout.List(ctx, ...)`) construct an `http.Request` and feed it to `client.Do(ctx, req)`.
3. **Rate Limiting**: `Do()` calls `rateLimiter.Wait(ctx)` first — blocks until a token is available from the 100 req/min bucket, or returns error if context is cancelled.
4. **HTTP Transport**: The internal `http.Client.Do(req)` fires. Auth header (`Bearer <token>`), `User-Agent` (`whoop-go/1.0.0`), and `Accept: application/json` are injected.
5. **429 Retry Loop**: On `429 Too Many Requests`, the body is drained, and backoff is computed. If `Retry-After` header exists, that value takes precedence over exponential backoff. Retry up to `maxRetries` times. Context cancellation during backoff is honored.
6. **Error Mapping**: Non-2xx responses are mapped through `mapHTTPError()` → `AuthError` (401/403), `RateLimitError` (429), or generic `APIError`.
7. **Deserialization**: Success bodies are decoded via `json.NewDecoder(resp.Body).Decode(&v)` into strongly-typed Go structs.

### Pagination Flow
- `List()` methods return a typed `*XxxPage` struct containing `Records []T` and `NextToken string`.
- Consumers call `page.NextPage(ctx)` to advance. It copies the original `ListOptions`, sets `NextToken`, and re-invokes `List()`.
- When `NextToken` is empty, `NextPage()` returns `ErrNoNextPage` (a sentinel error for `errors.Is()` checks).

## 4. Webhook Pipeline & Security
1. **Method Gate**: Only `POST` requests are accepted; all other methods return an error immediately.
2. **Header Extraction**: `X-Whoop-Signature` header is required; missing signature → immediate rejection.
3. **Bounds Protection**: `io.LimitReader(r.Body, 1<<20)` caps reads at 1MB to prevent OOM attacks.
4. **Single-Pass Hashing**: `io.TeeReader(limitedBody, mac)` feeds bytes simultaneously to the JSON decoder and the HMAC-SHA256 hasher. After decoding, remaining bytes are drained via `io.Copy(io.Discard, tee)` to ensure the full body is hashed.
5. **Signature Comparison**: The computed HMAC is `base64.StdEncoding` encoded and compared against the header value using `hmac.Equal()` (constant-time comparison to prevent timing attacks).
6. **JSON Error Deferral**: JSON decode errors are checked *after* signature validation, so even malformed-but-signed payloads get proper signature verification.

## 5. Concurrency Model
- **Token Bucket**: The `rateLimiter` uses `sync/atomic.Bool` for the enable/disable toggle and `golang.org/x/time/rate.Limiter` for the bucket itself. Both are safe for concurrent access.
- **URL Caching**: `CycleService`, `SleepService`, and `RecoveryService` use `sync.Once` to parse and cache their list endpoint URLs, preventing redundant allocations across goroutines.
- **Client Sharing**: A single `*whoop.Client` instance is designed to be shared across multiple goroutines. The `Do()` method clones the request (`req.Clone(ctx)`) to avoid mutation.

## 6. External Integrations
- **WHOOP API v2**: Base URL `https://api.prod.whoop.com/developer/v2`. OAuth endpoint at `https://api.prod.whoop.com/oauth/oauth2/auth` and `https://api.prod.whoop.com/oauth/oauth2/token`.
- **Dependencies**: Exactly one external dependency: `golang.org/x/time v0.14.0` for the token bucket rate limiter. Everything else is Go standard library.

## 7. Invariants & Red Lines
- **CRITICAL**: The `.whoop_token.json` file contains plaintext OAuth tokens. It is gitignored and MUST NEVER be committed.
- **CRITICAL**: The `io.LimitReader` cap of 1MB in `ParseWebhook()` MUST NOT be removed or increased without explicit security review.
- **CRITICAL**: `r.Body` MUST NOT be consumed before calling `ParseWebhook()` — the function relies on single-pass stream consumption via `TeeReader`.
- **CRITICAL**: Backoff base and max durations have defensive floors (`base <= 0` defaults to 1s, `max <= 0` defaults to 60s) to prevent negative or zero-duration sleeps.
- **CRITICAL**: The `Client.String()` and `Client.GoString()` methods redact the OAuth token. Do not add logging that bypasses these methods to print `c.token` directly.

## 8. Error Handling Strategy
- Standard Go `if err != nil` propagation throughout.
- Errors are wrapped with `fmt.Errorf("context: %w", err)` for unwrapping via `errors.Is()` and `errors.As()`.
- Three custom error types provide structured error handling: `*APIError`, `*RateLimitError`, `*AuthError`.
- The `mapHTTPError()` function truncates error bodies at 1000 characters to prevent log flooding from large error responses.
- Transient 429 errors are handled automatically by the retry loop; only exhausted retries surface the error to the consumer.

## 9. CI/CD Pipeline
- **GitHub Actions** (`.github/workflows/ci.yml`): Runs on push to `main` and pull requests.
  - `go mod tidy` → `go vet ./...` → `go test -v -race -cover ./...` → `golangci-lint` (v2.10.1, 5m timeout)
- **Pre-commit Hook** (`.githooks/pre-commit`): Runs `make lint` locally before every commit. Installed via `make setup`.

## 10. Local Development
- **Install & Setup**: `make setup` (configures local git hooks path to `.githooks/`)
- **Run Example**: `cp .env.example .env`, fill in `WHOOP_OAUTH_TOKEN` and `WHOOP_WEBHOOK_SECRET`, `source .env`, `make build-local && ./bin/example`
- **Get OAuth Token**: `export WHOOP_CLIENT_ID=... WHOOP_CLIENT_SECRET=... && go run cmd/auth/main.go`
- **Run Tests**: `make test` (`go test -v -race ./...`)
- **Run Coverage**: `make cover` (`go test -cover ./...`)
- **Lint**: `make lint` (`golangci-lint run ./...`)
- **Format**: `make tidy` (`go mod tidy && go fmt ./...`)
- **Vet**: `make vet` (`go vet ./...`)
- **Clean**: `make clean` (`rm -rf bin/`)
- **Cross-compile**: `make build-linux-amd64` or `make build-linux-arm64`

## 11. Environment Variables

### For `cmd/example/` (Webhook Listener)

| Variable | Required | Description |
|----------|----------|-------------|
| `WHOOP_OAUTH_TOKEN` | Yes | OAuth 2.0 access token (obtained via `cmd/auth/`) |
| `WHOOP_WEBHOOK_SECRET` | Yes | Secret key to validate incoming webhook HMAC signatures |

### For `cmd/auth/` (Token Generator)

| Variable | Required | Description |
|----------|----------|-------------|
| `WHOOP_CLIENT_ID` | Yes | OAuth 2.0 Client ID from the WHOOP Developer Portal |
| `WHOOP_CLIENT_SECRET` | Yes | OAuth 2.0 Client Secret from the WHOOP Developer Portal |
| `WHOOP_REDIRECT_URI` | No | OAuth callback URL (default: `http://localhost:8081/callback`) |