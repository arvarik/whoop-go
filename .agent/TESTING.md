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
# Follow the browser flow, then export the printed WHOOP_OAUTH_TOKEN
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
- **`mock_server_test.go`**: Contains `newMockServer(t)` which registers handlers for all domain endpoints (Cycle, Workout, Sleep, Recovery, User), plus error scenarios (429, 403, context cancellation delays). Also contains `newMockClient(ts, ...opts)` which creates a `*Client` pointed at the mock server.

**How it works:**
1. `newMockServer(t)` creates an `httptest.Server` with a `ServeMux` routing requests to literal JSON payload handlers.
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
GitHub Actions (`.github/workflows/ci.yml`) runs on every push to `main` and every PR:
1. `go mod tidy`
2. `go vet ./...`
3. `go test -v -race -cover ./...`
4. `golangci-lint` (v2.10.1, 5m timeout)

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

| Test File | What It Covers |
|-----------|----------------|
| `client_test.go` | Client bootstrapping, `Do()` error paths, retry exhaustion |
| `client_headers_test.go` | Auth header injection, User-Agent, Content-Type |
| `client_safety_test.go` | Token redaction in `String()` / `GoString()` |
| `options_test.go` | Functional Options (WithToken, WithBaseURL, etc.) |
| `ratelimit_test.go` | Token bucket enforcement, enable/disable toggle, backoff calculation with jitter |
| `pagination_test.go` | ListOptions encoding, NextPage iteration, ErrNoNextPage sentinel |
| `cycle_test.go` | CycleService GetByID and List with pagination |
| `workout_test.go` | WorkoutService GetByID and List with pagination |
| `sleep_test.go` | SleepService GetByID and List with pagination |
| `recovery_test.go` | RecoveryService GetByID and List with pagination |
| `profile_test.go` | UserService GetBasicProfile and GetBodyMeasurement |
| `webhook_test.go` | ParseWebhook signature validation, POST-only gate, missing headers |
| `security_test.go` | Payload size limits, tampered signatures, boundary attacks |
| `errors_test.go` | Error type mapping (APIError, AuthError, RateLimitError), Unwrap chains, message formatting |
| `example_test.go` | Godoc-compatible runnable examples for package documentation |
| `mock_server_test.go` | Shared test infrastructure: mock HTTP server and client factory |

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
| Race detector passes all packages | UNTESTED | |
| golangci-lint reports 0 issues | UNTESTED | |

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
| golangci-lint produces 0 issues | _YYYY-MM-DD_ | `make lint` |