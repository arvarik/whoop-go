# Testing Strategy & Results

_This file tracks test methods, scenarios, and results with concrete execution evidence. Bugs found here block the release of a feature. Agents must update this during the Test and Fix phases._

## 0. Local Development Setup

### Prerequisites
- Go 1.24+
- `golangci-lint` installed (`brew install golangci-lint` or equivalent)

### Start the Example App
```bash
cp .env.example .env
# Fill in WHOOP_CLIENT_ID, WHOOP_CLIENT_SECRET, WHOOP_WEBHOOK_SECRET
source .env
make build-local
./bin/example
```

### Git Hooks
Run this immediately after cloning to configure the local Git hooks for linting on commit:
```bash
make setup
```

## 1. Test Methods & Tools

### Unit / Integration Tests
- **Run all tests (with race detector)**: `make test` (Executes `go test -v -race ./...`)
- **Run coverage**: `make cover` (Executes `go test -cover ./...`)

### Type Checking & Linting
- **Linting**: `make lint` (Executes `golangci-lint run ./...`). This must produce 0 issues.
- **Formatting**: `make tidy` (Executes `go mod tidy` and `go fmt ./...`).
- **Vet**: `make vet` (Executes `go vet ./...`).

## 2. Execution Evidence Rules
_Never mark a test as PASS without evidence._
- For Go tests, paste the output of `go test -v -race ./...` (showing individual test PASS/FAIL lines) into the Notes column.
- For type checking / linting, paste the command and its output (e.g., `make lint → 0 issues`).
- "PASS" with no evidence is treated as UNTESTED.

---

## Current Feature Scenarios: Bootstrapping

| Scenario | Status | Notes (Evidence) |
|----------|--------|------------------|
| Empty/null/missing inputs | UNTESTED | |
| Valid payload creates resource | UNTESTED | |
| Invalid payload returns structured error | UNTESTED | |
| State transitions | UNTESTED | |

## Bugs Found (Fix Phase Queue)
_List specific bugs discovered during testing. Agents in the 'Fix' phase will read this section._
- (None currently)

---

## Regression Scenarios (Persistent)
_These scenarios survive the Ship phase cleanup. They are re-run on every release to catch regressions. Add critical paths and previously-shipped bug fixes here._

| Scenario | Last Verified | Notes |
|----------|---------------|-------|
| _Example: Race detector passes on all packages_ | _YYYY-MM-DD_ | _`make test`_ |