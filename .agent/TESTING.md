# Testing Strategy & Results

_This file tracks test methods, scenarios, and results with concrete execution evidence. Bugs found here block the release of a feature. Agents must update this during the Test and Fix phases._

## 0. Local Development Setup

### Prerequisites
- Go 1.24+
- `golangci-lint` installed (`brew install golangci-lint` or equivalent)

### Start the Reference Webhook Listener
```bash
cp .env.example .env
# Fill in WHOOP_OAUTH_TOKEN (from cmd/auth) and WHOOP_WEBHOOK_SECRET
source .env
make build-local
./bin/example
```

### Get an OAuth Token
```bash
export WHOOP_CLIENT_ID="your-client-id"
export WHOOP_CLIENT_SECRET="your-client-secret"
go run cmd/auth/main.go
# First run: Follow the browser flow, authorize, and the session is saved to .whoop_token.json
# Subsequent runs: Token is auto-refreshed from the saved session — no browser login needed
```

### Git Hooks
Run this immediately after cloning to configure pre-commit linting:
```bash
make setup
```

## 1. Test Methods & Tools

### Unit Tests — The Mock Multiplexer Pattern
Tests do NOT mock the `whoop.Client` via interfaces. Instead, the project uses `net/http/httptest.Server` to create an ephemeral HTTP server with hand-crafted route handlers.

**Key test infrastructure files:**
- **`whoop/mock_server_test.go`**: Contains `newMockServer(t)` which registers handlers for all domain endpoints (Cycle, Workout, Sleep, Recovery, User), plus error scenarios (429, 403, context cancellation delays). Also contains `newMockClient(ts, ...opts)` which creates a `*Client` pointed at the mock server with shorter backoff defaults.

**How it works:**
1. `newMockServer(t)` creates an `httptest.Server` with a `ServeMux` routing requests to literal JSON payload handlers (12 handlers total: Cycle GetByID, Workout List/GetByID, Sleep List/GetByID, Recovery List/GetByID, User Profile, User BodyMeasurement, 429 generator, 403 generator, delay endpoint).
2. `newMockClient(ts)` calls `whoop.NewClient(whoop.WithBaseURL(ts.URL), ...)` to point the real client at the fake server.
3. Tests exercise the full request pipeline: Functional Options → `Do()` → rate limiter → HTTP transport → JSON decode → typed structs.

### Running Tests
- **All tests with race detector**: `make test` → `go test -v -race ./...`
- **Coverage report**: `make cover` → `go test -cover ./...`
- **Race detector is NON-NEGOTIABLE**: Every test run uses `-race`. No exceptions.

### Type Checking & Linting
- **Lint**: `make lint` → `golangci-lint run ./...` → MUST produce 0 issues.
- **Format**: `make tidy` → `go mod tidy && go fmt ./...`
- **Vet**: `make vet` → `go vet ./...`

### CI Pipeline
GitHub Actions (`.github/workflows/ci.yml`) runs on every push to `main` and every PR to `main`:
1. `actions/checkout@v4`
2. `actions/setup-go@v5` with `go-version-file: go.mod`
3. `go mod tidy`
4. `go vet ./...`
5. `go test -v -race -cover ./...`
6. `golangci-lint` via `golangci/golangci-lint-action@v7` (v2.10.1, 5m timeout)

## 2. Execution Evidence Rules (CRITICAL FOR AGENTS)
_Never mark a test as PASS without evidence. Humans rely on this file to verify agent accuracy._

### Required Evidence Format
- **Go tests**: Paste the raw output of `go test -v -race ./...` showing individual `--- PASS` / `--- FAIL` lines and the final `ok` summary with coverage percentage.
- **Linting**: Paste the command and its output (e.g., `make lint → 0 issues`).
- **"PASS" with no evidence is treated as UNTESTED** — a blocking failure.

### Example Evidence Entry
```
$ go test -v -race ./...
=== RUN   TestCycleGetByID
--- PASS: TestCycleGetByID (0.00s)
=== RUN   TestWebhookInvalidSignature
--- PASS: TestWebhookInvalidSignature (0.00s)
ok  	github.com/arvarik/whoop-go/whoop	0.342s
```

---

## 3. Test File Inventory

### Library Tests (`whoop/` package)

| Test File | Package | What It Covers |
|-----------|---------|----------------|
| `mock_server_test.go` | `whoop` | Shared test infrastructure: `newMockServer(t)` with 12 route handlers, `newMockClient(ts, ...opts)` factory |
| `client_test.go` | `whoop` | `Do()` context cancellation, `Do()` error mapping (403 → `*AuthError`), `Client.String()` / `GoString()` token redaction across `%v`, `%+v`, `%#v`, `%s` format verbs |
| `client_headers_test.go` | `whoop` | Auth header injection with/without token, `Accept`, `User-Agent`, `Content-Type` defaults for GET/POST, custom Content-Type preservation |
| `client_safety_test.go` | `whoop` | Verifies `req.Clone(ctx)` prevents header mutation on the caller's original request — uses a `safetyCheckTransport` mock to intercept without network calls |
| `options_test.go` | `whoop` | Functional Options (`WithToken`, `WithBaseURL`, `WithMaxRetries`, `WithBackoffBase`, `WithBackoffMax`, `WithHTTPClient`, `WithRateLimiting`) |
| `ratelimit_test.go` | `whoop` | Token bucket enforcement, enable/disable toggle via `SetAutoLimiting`, `calculateBackoff()` exponential growth, jitter bounds, defensive floors |
| `pagination_test.go` | `whoop` | `ListOptions.encode()` query parameter generation, `NextPage` iteration, `ErrNoNextPage` sentinel |
| `cycle_test.go` | `whoop` | `CycleService.GetByID` and `List` with pagination |
| `workout_test.go` | `whoop` | `WorkoutService.GetByID` and `List` with pagination |
| `sleep_test.go` | `whoop` | `SleepService.GetByID` and `List` with pagination |
| `recovery_test.go` | `whoop` | `RecoveryService.GetByID` and `List` with pagination |
| `profile_test.go` | `whoop` | `UserService.GetBasicProfile` and `GetBodyMeasurement` |
| `webhook_test.go` | `whoop` | `ParseWebhook` valid signature, invalid signature, missing header, wrong HTTP method (non-POST), oversized body (>1MB), invalid JSON with valid signature, empty body, pre-consumed body |
| `security_test.go` | `whoop` | `mapHTTPError()` body truncation at 1000 chars: large body (2000 chars → 1003 with `...`), short body (unchanged), exactly 1000 chars (no truncation) |
| `errors_test.go` | `whoop` | Error type mapping (`APIError`, `AuthError`, `RateLimitError`), `Unwrap()` chains, message formatting, `errors.Is()`/`errors.As()` compatibility |
| `example_test.go` | `whoop_test` | Godoc-compatible runnable examples for `NewClient`, `UserService`, `CycleService`, `WorkoutService`, `SleepService`, `RecoveryService`, `ParseWebhook` |

### Executable Tests (`cmd/` packages)

| Test File | Package | What It Covers |
|-----------|---------|----------------|
| `cmd/example/main_test.go` | `main` | Webhook handler integration: valid `workout.updated` event → queued, non-workout event → not queued, invalid signature → 401, full job queue → graceful drop. Uses table-driven subtests with `signPayload()` helper. |

---

## 4. Current Feature Scenarios

| Scenario | Status | Notes (Evidence) |
|----------|--------|------------------|

---

## Backend Route Coverage Matrix

_Populated by the SDET during the Trap phase. One row per exported function or API method. All cells must show PASS with execution evidence or FAIL with reproduction steps._

| Endpoint / Function | Method | Valid Input | Invalid Input | Error Handling | Edge Cases |
|---------------------|--------|-------------|---------------|----------------|------------|
| `NewClient` | `NewClient(opts ...Option)` | PASS `go test -v` | N/A | N/A | N/A |
| `Client.Do` | `Do(ctx, req)` | PASS `go test -v` | N/A | PASS `go test -v` | PASS `go test -v` |
| `Client.Get` | `Get(ctx, path, v)` | PASS `go test -v` | N/A | PASS `go test -v` | N/A |
| `Client.String` | `String()` | PASS `go test -v` | N/A | N/A | PASS `go test -v` |
| `Client.GoString` | `GoString()` | PASS `go test -v` | N/A | N/A | PASS `go test -v` |
| `WithToken` | `WithToken(token)` | PASS `go test -v` | N/A | N/A | N/A |
| `WithBaseURL` | `WithBaseURL(url)` | PASS `go test -v` | N/A | N/A | N/A |
| `WithHTTPClient` | `WithHTTPClient(c)` | PASS `go test -v` | N/A | N/A | N/A |
| `WithMaxRetries` | `WithMaxRetries(retries)` | PASS `go test -v` | N/A | N/A | N/A |
| `WithBackoffBase` | `WithBackoffBase(base)` | PASS `go test -v` | N/A | N/A | N/A |
| `WithBackoffMax` | `WithBackoffMax(max)` | PASS `go test -v` | N/A | N/A | N/A |
| `WithRateLimiting` | `WithRateLimiting(enabled)` | PASS `go test -v` | N/A | N/A | N/A |
| `CycleService.GetByID` | `GetByID(ctx, id int)` | PASS `go test -v` | N/A | PASS `go test -v` | N/A |
| `CycleService.List` | `List(ctx, *ListOptions)` | PASS `go test -v` | N/A | PASS `go test -v` | PASS `go test -v` |
| `CyclePage.NextPage` | `NextPage(ctx)` | PASS `go test -v` | N/A | PASS `go test -v` | PASS `go test -v` |
| `WorkoutService.GetByID` | `GetByID(ctx, id string)` | PASS `go test -v` | N/A | PASS `go test -v` | N/A |
| `WorkoutService.List` | `List(ctx, *ListOptions)` | PASS `go test -v` | N/A | PASS `go test -v` | PASS `go test -v` |
| `WorkoutPage.NextPage` | `NextPage(ctx)` | PASS `go test -v` | N/A | PASS `go test -v` | PASS `go test -v` |
| `SleepService.GetByID` | `GetByID(ctx, id string)` | PASS `go test -v` | N/A | PASS `go test -v` | N/A |
| `SleepService.List` | `List(ctx, *ListOptions)` | PASS `go test -v` | N/A | PASS `go test -v` | PASS `go test -v` |
| `SleepPage.NextPage` | `NextPage(ctx)` | PASS `go test -v` | N/A | PASS `go test -v` | PASS `go test -v` |
| `RecoveryService.GetByID` | `GetByID(ctx, cycleID int)` | PASS `go test -v` | N/A | PASS `go test -v` | N/A |
| `RecoveryService.List` | `List(ctx, *ListOptions)` | PASS `go test -v` | N/A | PASS `go test -v` | PASS `go test -v` |
| `RecoveryPage.NextPage` | `NextPage(ctx)` | PASS `go test -v` | N/A | PASS `go test -v` | PASS `go test -v` |
| `UserService.GetBasicProfile` | `GetBasicProfile(ctx)` | PASS `go test -v` | N/A | PASS `go test -v` | N/A |
| `UserService.GetBodyMeasurement` | `GetBodyMeasurement(ctx)` | PASS `go test -v` | N/A | PASS `go test -v` | N/A |
| `ParseWebhook` | `ParseWebhook(r, secret)` | PASS `go test -v` | PASS `go test -v` | PASS `go test -v` | PASS `go test -v` |
| `APIError.Error` | `Error()` | PASS `go test -v` | N/A | N/A | N/A |
| `APIError.Unwrap` | `Unwrap()` | PASS `go test -v` | N/A | N/A | N/A |
| `RateLimitError.Error` | `Error()` | PASS `go test -v` | N/A | N/A | PASS `go test -v` |
| `RateLimitError.Unwrap` | `Unwrap()` | PASS `go test -v` | N/A | N/A | N/A |
| `AuthError.Error` | `Error()` | PASS `go test -v` | N/A | N/A | N/A |
| `AuthError.Unwrap` | `Unwrap()` | PASS `go test -v` | N/A | N/A | N/A |

---

## Frontend Component State Matrix

N/A — Frontend topology is not active for this project.

---

## ML / AI Evaluation Thresholds

N/A — ML/AI topology is not active for this project.

## 5. Bugs Found (Fix Phase Queue)
_List specific bugs discovered during testing. Agents in the 'Fix' phase will read this section._
- (None currently)

---

## 6. Regression Scenarios (Persistent)
_These scenarios survive the Ship phase cleanup. They are re-run on every release to catch regressions._

| Scenario | Last Verified | Notes |
|----------|---------------|-------|
| Race detector passes on all packages | 2026-04-20 | `make test` |
| HMAC signature validates correctly signed bodies | 2026-04-20 | `make test` |
| HMAC signature rejects tampered bodies | 2026-04-20 | `make test` |
| Payload >1MB is rejected by ParseWebhook | 2026-04-20 | `make test` |
| Token redacted in all string representations | 2026-04-20 | `make test` |
| Request clone prevents original header mutation | 2026-04-20 | `make test` |
| Error body truncation at 1000 chars | 2026-04-20 | `make test` |
| golangci-lint produces 0 issues | 2026-04-20 | `make lint` |

### Backend Route Coverage

| Route | Method | Auth | Contract Test | Integration Test |
|-------|--------|------|---------------|------------------|
| _Fill in your routes_ | | | | |

### API Surface Coverage

| Export | Unit Test | Type Test | Doc Example | Breaking Change Guard |
|-------|-----------|-----------|-------------|----------------------|
| _Fill in your exports_ | | | | |

