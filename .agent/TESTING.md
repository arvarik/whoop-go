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
| Client bootstrapping with defaults | UNTESTED | |
| Functional Options override defaults | UNTESTED | |
| Auth header injected on requests | UNTESTED | |
| Token redacted in String()/GoString() | UNTESTED | |
| Rate limiter enforces 100 req/min bucket | UNTESTED | |
| Backoff calculation with full jitter | UNTESTED | |
| HTTP 429 retry loop with Retry-After | UNTESTED | |
| HTTP 429 retry exhaustion returns RateLimitError | UNTESTED | |
| HTTP 401/403 returns AuthError | UNTESTED | |
| Context cancellation during backoff | UNTESTED | |
| Paginated List → NextPage → ErrNoNextPage | UNTESTED | |
| Webhook POST-only gate | UNTESTED | |
| Webhook missing X-Whoop-Signature | UNTESTED | |
| Webhook payload exceeds 1MB limit | UNTESTED | |
| Webhook invalid signature rejection | UNTESTED | |
| Webhook valid signature + JSON decode | UNTESTED | |
| Webhook invalid JSON with valid signature | UNTESTED | |
| Webhook empty body | UNTESTED | |
| Webhook pre-consumed body | UNTESTED | |
| Error body truncation at 1000 chars | UNTESTED | |
| Request clone prevents header mutation | UNTESTED | |
| Content-Type set only for non-GET requests | UNTESTED | |
| Race detector passes all packages | UNTESTED | |
| golangci-lint reports 0 issues | UNTESTED | |

---

## Backend Route Coverage Matrix

_Populated by the SDET during the Trap phase. One row per exported function or API method. All cells must show PASS with execution evidence or FAIL with reproduction steps._

| Endpoint / Function | Method | Valid Input | Invalid Input | Error Handling | Edge Cases |
|---------------------|--------|-------------|---------------|----------------|------------|
| `NewClient` | `NewClient(opts ...Option)` | | | | |
| `Client.Do` | `Do(ctx, req)` | | | | |
| `Client.Get` | `Get(ctx, path, v)` | | | | |
| `Client.String` | `String()` | | | | |
| `Client.GoString` | `GoString()` | | | | |
| `WithToken` | `WithToken(token)` | | | | |
| `WithBaseURL` | `WithBaseURL(url)` | | | | |
| `WithHTTPClient` | `WithHTTPClient(c)` | | | | |
| `WithMaxRetries` | `WithMaxRetries(retries)` | | | | |
| `WithBackoffBase` | `WithBackoffBase(base)` | | | | |
| `WithBackoffMax` | `WithBackoffMax(max)` | | | | |
| `WithRateLimiting` | `WithRateLimiting(enabled)` | | | | |
| `CycleService.GetByID` | `GetByID(ctx, id int)` | | | | |
| `CycleService.List` | `List(ctx, *ListOptions)` | | | | |
| `CyclePage.NextPage` | `NextPage(ctx)` | | | | |
| `WorkoutService.GetByID` | `GetByID(ctx, id string)` | | | | |
| `WorkoutService.List` | `List(ctx, *ListOptions)` | | | | |
| `WorkoutPage.NextPage` | `NextPage(ctx)` | | | | |
| `SleepService.GetByID` | `GetByID(ctx, id string)` | | | | |
| `SleepService.List` | `List(ctx, *ListOptions)` | | | | |
| `SleepPage.NextPage` | `NextPage(ctx)` | | | | |
| `RecoveryService.GetByID` | `GetByID(ctx, cycleID int)` | | | | |
| `RecoveryService.List` | `List(ctx, *ListOptions)` | | | | |
| `RecoveryPage.NextPage` | `NextPage(ctx)` | | | | |
| `UserService.GetBasicProfile` | `GetBasicProfile(ctx)` | | | | |
| `UserService.GetBodyMeasurement` | `GetBodyMeasurement(ctx)` | | | | |
| `ParseWebhook` | `ParseWebhook(r, secret)` | | | | |
| `APIError.Error` | `Error()` | | | | |
| `APIError.Unwrap` | `Unwrap()` | | | | |
| `RateLimitError.Error` | `Error()` | | | | |
| `RateLimitError.Unwrap` | `Unwrap()` | | | | |
| `AuthError.Error` | `Error()` | | | | |
| `AuthError.Unwrap` | `Unwrap()` | | | | |

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
| Race detector passes on all packages | _YYYY-MM-DD_ | `make test` |
| HMAC signature validates correctly signed bodies | _YYYY-MM-DD_ | `make test` |
| HMAC signature rejects tampered bodies | _YYYY-MM-DD_ | `make test` |
| Payload >1MB is rejected by ParseWebhook | _YYYY-MM-DD_ | `make test` |
| Token redacted in all string representations | _YYYY-MM-DD_ | `make test` |
| Request clone prevents original header mutation | _YYYY-MM-DD_ | `make test` |
| Error body truncation at 1000 chars | _YYYY-MM-DD_ | `make test` |
| golangci-lint produces 0 issues | _YYYY-MM-DD_ | `make lint` |