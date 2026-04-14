# Architecture

_This document acts as the definitive anchor for understanding system design, data models, API contracts, and technology boundaries. Update this document during the Design and Review phases._

## 1. Tech Stack & Infrastructure
- **Language / Runtime**: Go 1.24.0+ (uses `math/rand/v2` which requires Go 1.22+)
- **Frontend / UI**: None (Go SDK Library)
- **Backend / Database**: None
- **Deployment**: Single library archive imported via standard Go modules (`go get github.com/arvarik/whoop-go/whoop`)
- **Package Management**: Go modules (`go.mod`) + Makefile for local automation
- **Build System**: Standard `go build`

## 2. Directory Structure & Internal Package Mapping

### Core Library (`whoop/`)
The flat `whoop/` package is the single importable unit. All source files live at the top level—no sub-packages.

| File | Role |
|------|------|
| `client.go` | Core `Client` struct, `Do()` method (authentication, rate limiting, retry loop with 4096-byte body drains), `Get()` convenience helper. Implements `fmt.Stringer` and `fmt.GoStringer` to redact tokens in logs. Conditionally sets `Content-Type: application/json` on non-GET requests when no Content-Type is already present. |
| `options.go` | Functional Options pattern: `WithToken()`, `WithBaseURL()`, `WithHTTPClient()`, `WithMaxRetries()`, `WithBackoffBase()`, `WithBackoffMax()`, `WithRateLimiting()`. Options set values directly with no validation—defensive floors for backoff values are enforced in `calculateBackoff()`, not in the Option functions. |
| `ratelimit.go` | Thread-safe token bucket rate limiter (`golang.org/x/time/rate`) configured for 100 req/min with burst of 100. Uses `atomic.Bool` for toggling. Contains `calculateBackoff()` with exponential backoff and full jitter via `math/rand/v2`. Defensive floors: `base <= 0` defaults to 1s, `max <= 0` defaults to 60s. |
| `pagination.go` | `ListOptions` struct (`Limit`, `Start`, `End`, `NextToken`), URL query encoder via `encode(*url.URL)`, `nextPageOpts()` copy helper, and generic `paginatedResponse[T any]` type using Go generics. `getPaginated[T]()` copies the URL before encoding to avoid mutating cached base URLs. |
| `webhooks.go` | `ParseWebhook()`: memory-capped `io.LimitReader` (1MB via `maxWebhookBodySize = 1 << 20`) → `io.TeeReader` → `crypto/hmac` SHA-256 → `base64.StdEncoding` signature comparison. Returns `*WebhookEvent` (skinny payload with `UserID`, `ID`, `Type`, `TraceID`). Webhook errors are plain `errors.New()` values, not typed errors. |
| `errors.go` | Three typed errors: `APIError` (generic HTTP errors with `StatusCode`, `Message`, `URL`, `Err`), `RateLimitError` (429 with `RetryAfter int` in seconds and `Err error`), `AuthError` (401/403 with `StatusCode`, `Message`, `Err error`). All implement `Unwrap()` for `errors.Is()`/`errors.As()`. `mapHTTPError()` dispatches by status code and truncates error bodies at 1000 characters. |
| `scopes.go` | OAuth 2.0 scope constants (`ScopeOffline`, `ScopeReadRecovery`, `ScopeReadCycles`, `ScopeReadSleep`, `ScopeReadWorkout`, `ScopeReadProfile`, `ScopeReadBodyMeasurement`) as the `Scope` type (underlying `string`). |
| `doc.go` | Package-level godoc with Quick Start, Pagination, and Webhook examples. |

### Domain Services & Types
Each domain maps 1:1 to a WHOOP API resource:

| File | Service | Key Types | Methods |
|------|---------|-----------|---------|
| `cycle.go` | `CycleService` | `Cycle`, `Score`, `CyclePage` | `GetByID(ctx, id int)`, `List(ctx, *ListOptions)`, `CyclePage.NextPage(ctx)` |
| `workout.go` | `WorkoutService` | `Workout`, `WorkoutScore`, `ZoneDurations`, `WorkoutPage` | `GetByID(ctx, id string)`, `List(ctx, *ListOptions)`, `WorkoutPage.NextPage(ctx)` |
| `sleep.go` | `SleepService` | `Sleep`, `SleepScore`, `StageSummary`, `SleepNeeded`, `SleepPage` | `GetByID(ctx, id string)`, `List(ctx, *ListOptions)`, `SleepPage.NextPage(ctx)` |
| `recovery.go` | `RecoveryService` | `Recovery`, `RecoveryScore`, `RecoveryPage` | `GetByID(ctx, cycleID int)`, `List(ctx, *ListOptions)`, `RecoveryPage.NextPage(ctx)` |
| `profile.go` | `UserService` | `BasicProfile`, `BodyMeasurement` | `GetBasicProfile(ctx)`, `GetBodyMeasurement(ctx)` |

> **Note**: `Recovery.GetByID()` takes a `cycleID` (not a generic resource ID) because recoveries are always fetched relative to a cycle via `/cycle/{cycleID}/recovery`.

### Complete Struct Field Reference

#### `Cycle` (`cycle.go`)
| Field | Type | JSON | Notes |
|-------|------|------|-------|
| `ID` | `int` | `id` | |
| `UserID` | `int` | `user_id` | |
| `CreatedAt` | `time.Time` | `created_at` | |
| `UpdatedAt` | `time.Time` | `updated_at` | |
| `Start` | `time.Time` | `start` | |
| `End` | `*time.Time` | `end` | Pointer — active cycles have no end |
| `TimezoneOffset` | `string` | `timezone_offset` | |
| `ScoreState` | `string` | `score_state` | |
| `Score` | `*Score` | `score,omitempty` | nil when `score_state != "SCORED"` |

#### `Score` (Cycle Score, `cycle.go`)
| Field | Type | JSON |
|-------|------|------|
| `Strain` | `float64` | `strain` |
| `Kilojoule` | `float64` | `kilojoule` |
| `AverageHeartRate` | `int` | `average_heart_rate` |
| `MaxHeartRate` | `int` | `max_heart_rate` |

#### `Workout` (`workout.go`)
| Field | Type | JSON | Notes |
|-------|------|------|-------|
| `ID` | `string` | `id` | UUID (v2 API) |
| `V1ID` | `*int` | `v1_id,omitempty` | Legacy v1 numeric ID, optional |
| `UserID` | `int` | `user_id` | |
| `CreatedAt` | `time.Time` | `created_at` | |
| `UpdatedAt` | `time.Time` | `updated_at` | |
| `Start` | `time.Time` | `start` | |
| `End` | `time.Time` | `end` | |
| `TimezoneOffset` | `string` | `timezone_offset` | |
| `SportID` | `int` | `sport_id` | |
| `SportName` | `string` | `sport_name` | |
| `ScoreState` | `string` | `score_state` | |
| `Score` | `*WorkoutScore` | `score,omitempty` | nil when unscored |

#### `WorkoutScore` (`workout.go`)
| Field | Type | JSON |
|-------|------|------|
| `Strain` | `float64` | `strain` |
| `AverageHeartRate` | `int` | `average_heart_rate` |
| `MaxHeartRate` | `int` | `max_heart_rate` |
| `Kilojoule` | `float64` | `kilojoule` |
| `PercentRecorded` | `float64` | `percent_recorded` |
| `DistanceMeter` | `*float64` | `distance_meter` |
| `AltitudeGainMeter` | `*float64` | `altitude_gain_meter` |
| `AltitudeChangeMeter` | `*float64` | `altitude_change_meter` |
| `ZoneDuration` | `*ZoneDurations` | `zone_durations` |

#### `ZoneDurations` (`workout.go`)
| Field | Type | JSON |
|-------|------|------|
| `ZoneZeroMilli` | `int` | `zone_zero_milli` |
| `ZoneOneMilli` | `int` | `zone_one_milli` |
| `ZoneTwoMilli` | `int` | `zone_two_milli` |
| `ZoneThreeMilli` | `int` | `zone_three_milli` |
| `ZoneFourMilli` | `int` | `zone_four_milli` |
| `ZoneFiveMilli` | `int` | `zone_five_milli` |

#### `Sleep` (`sleep.go`)
| Field | Type | JSON | Notes |
|-------|------|------|-------|
| `ID` | `string` | `id` | UUID (v2 API) |
| `CycleID` | `int` | `cycle_id` | |
| `V1ID` | `*int` | `v1_id,omitempty` | Legacy v1 numeric ID, optional |
| `UserID` | `int` | `user_id` | |
| `CreatedAt` | `time.Time` | `created_at` | |
| `UpdatedAt` | `time.Time` | `updated_at` | |
| `Start` | `time.Time` | `start` | |
| `End` | `time.Time` | `end` | |
| `TimezoneOffset` | `string` | `timezone_offset` | |
| `Nap` | `bool` | `nap` | |
| `ScoreState` | `string` | `score_state` | |
| `Score` | `*SleepScore` | `score,omitempty` | nil when unscored |

#### `SleepScore` (`sleep.go`)
| Field | Type | JSON |
|-------|------|------|
| `StageSummary` | `*StageSummary` | `stage_summary` |
| `SleepNeeded` | `*SleepNeeded` | `sleep_needed` |
| `RespiratoryRate` | `float64` | `respiratory_rate` |
| `SleepPerformancePercentage` | `float64` | `sleep_performance_percentage` |
| `SleepConsistencyPercentage` | `float64` | `sleep_consistency_percentage` |
| `SleepEfficiencyPercentage` | `float64` | `sleep_efficiency_percentage` |

#### `StageSummary` (`sleep.go`)
| Field | Type | JSON |
|-------|------|------|
| `TotalInBedTimeMilli` | `int` | `total_in_bed_time_milli` |
| `TotalAwakeTimeMilli` | `int` | `total_awake_time_milli` |
| `TotalNoDataTimeMilli` | `int` | `total_no_data_time_milli` |
| `TotalLightSleepTimeMilli` | `int` | `total_light_sleep_time_milli` |
| `TotalSlowWaveSleepTimeMilli` | `int` | `total_slow_wave_sleep_time_milli` |
| `TotalRemSleepTimeMilli` | `int` | `total_rem_sleep_time_milli` |
| `SleepCycleCount` | `int` | `sleep_cycle_count` |
| `DisturbanceCount` | `int` | `disturbance_count` |

#### `SleepNeeded` (`sleep.go`)
| Field | Type | JSON |
|-------|------|------|
| `BaselineMilli` | `int` | `baseline_milli` |
| `NeedFromSleepDebtMilli` | `int` | `need_from_sleep_debt_milli` |
| `NeedFromRecentStrainMilli` | `int` | `need_from_recent_strain_milli` |
| `NeedFromRecentNapMilli` | `int` | `need_from_recent_nap_milli` |

#### `Recovery` (`recovery.go`)
| Field | Type | JSON |
|-------|------|------|
| `CycleID` | `int` | `cycle_id` |
| `SleepID` | `string` | `sleep_id` |
| `UserID` | `int` | `user_id` |
| `CreatedAt` | `time.Time` | `created_at` |
| `UpdatedAt` | `time.Time` | `updated_at` |
| `ScoreState` | `string` | `score_state` |
| `Score` | `*RecoveryScore` | `score,omitempty` |

#### `RecoveryScore` (`recovery.go`)
| Field | Type | JSON |
|-------|------|------|
| `UserCalibrating` | `bool` | `user_calibrating` |
| `RecoveryScore` | `float64` | `recovery_score` |
| `RestingHeartRate` | `float64` | `resting_heart_rate` |
| `HrvRmssdMilli` | `float64` | `hrv_rmssd_milli` |
| `Spo2Percentage` | `float64` | `spo2_percentage` |
| `SkinTempCelsius` | `float64` | `skin_temp_celsius` |

#### `BasicProfile` (`profile.go`)
| Field | Type | JSON |
|-------|------|------|
| `UserID` | `int` | `user_id` |
| `Email` | `string` | `email` |
| `FirstName` | `string` | `first_name` |
| `LastName` | `string` | `last_name` |

#### `BodyMeasurement` (`profile.go`)
| Field | Type | JSON |
|-------|------|------|
| `HeightMeter` | `float64` | `height_meter` |
| `WeightKilogram` | `float64` | `weight_kilogram` |
| `MaxHeartRate` | `int` | `max_heart_rate` |

#### `WebhookEvent` (`webhooks.go`)
| Field | Type | JSON |
|-------|------|------|
| `UserID` | `int` | `user_id` |
| `ID` | `string` | `id` |
| `Type` | `string` | `type` |
| `TraceID` | `string` | `trace_id` |

#### `ListOptions` (`pagination.go`)
| Field | Type | URL Tag | Notes |
|-------|------|---------|-------|
| `Limit` | `int` | `limit,omitempty` | API max is 50 |
| `Start` | `*time.Time` | `start,omitempty` | ISO-8601 / RFC3339 |
| `End` | `*time.Time` | `end,omitempty` | ISO-8601 / RFC3339 |
| `NextToken` | `string` | `nextToken,omitempty` | Managed by paginator |

### Executables (`cmd/`)
- **`cmd/example/`**: Reference webhook listener app. Demonstrates `ParseWebhook()`, worker pool pattern (5 goroutines, buffered channel of 100), and REST API pulls triggered by skinny webhook events. Uses `WHOOP_OAUTH_TOKEN` and `WHOOP_WEBHOOK_SECRET` env vars. Includes `main_test.go` with table-driven tests covering valid/invalid signatures, event type filtering, and queue-full backpressure behavior.
- **`cmd/auth/`**: Standalone OAuth 2.0 Authorization Code flow helper. Starts a local HTTP server, handles the browser callback, exchanges the auth code for tokens, saves to `.whoop_token.json`, and supports **automatic token refresh** — subsequent runs detect the saved session and silently refresh without opening a browser. Uses `WHOOP_CLIENT_ID` and `WHOOP_CLIENT_SECRET` env vars.

### Supporting Directories
- **`docs/`**: Contains `archive/`, `designs/`, `explorations/`, `plans/` subdirectories for design documents.
- **`.githooks/`**: Contains a `pre-commit` hook that runs `make lint` before every commit.
- **`.github/workflows/`**: Contains `ci.yml` — GitHub Actions CI pipeline (tidy, vet, test with race/cover, golangci-lint).
- **`bin/`**: Compiled output directory (gitignored).

## 3. System Boundaries & Data Flow

### Request Lifecycle
1. **Bootstrapping**: Consumer invokes `whoop.NewClient(whoop.WithToken("..."))`. Options configure the internal `http.Client` (default 30s timeout), backoff (base: 1s, max: 60s), retries (default: 3), and rate limiter (enabled by default, 100 req/min with burst of 100).
2. **Request Execution**: Service methods (e.g., `client.Workout.List(ctx, ...)`) construct an `http.Request` and feed it to `client.Do(ctx, req)`.
3. **Request Cloning**: `Do()` calls `req.Clone(ctx)` to prevent mutation of the caller's original request object.
4. **Header Injection**: On the cloned request, the following headers are set:
   - `Authorization: Bearer <token>` (only if token is non-empty)
   - `Accept: application/json` (always)
   - `User-Agent: whoop-go/1.0.0` (always)
   - `Content-Type: application/json` (only for non-GET requests when no Content-Type is already set)
5. **Rate Limiting**: `Do()` calls `rateLimiter.Wait(ctx)` — blocks until a token is available from the 100 req/min bucket, or returns error if context is cancelled.
6. **HTTP Transport**: The internal `http.Client.Do(req)` fires.
7. **429 Retry Loop**: On `429 Too Many Requests`, the body is drained via `io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))` (4KB cap to prevent memory exhaustion during drains), and backoff is computed. If `Retry-After` header exists and parses to a positive integer, that value (in seconds) takes precedence over exponential backoff. Retry up to `maxRetries` times. Context cancellation during backoff is honored via `select` on `ctx.Done()`.
8. **Error Mapping**: Non-2xx responses (status >= 400) have their bodies read via `io.ReadAll(io.LimitReader(resp.Body, 4096))` and mapped through `mapHTTPError()` → `AuthError` (401/403), `RateLimitError` (429), or generic `APIError`.
9. **Deserialization**: Success bodies are decoded via `json.NewDecoder(resp.Body).Decode(&v)` into strongly-typed Go structs. Body close errors are captured via named return and deferred close.

### Pagination Flow
- `List()` methods return a typed `*XxxPage` struct containing `Records []T` and `NextToken string`.
- Consumers call `page.NextPage(ctx)` to advance. It copies the original `ListOptions` via `nextPageOpts()`, sets `NextToken`, and re-invokes `List()`.
- When `NextToken` is empty, `NextPage()` returns `ErrNoNextPage` (a sentinel error for `errors.Is()` checks).
- The `getPaginated[T]()` helper copies the cached URL before encoding query parameters to avoid mutating the `sync.Once`-cached base URL.

## 4. Webhook Pipeline & Security
1. **Method Gate**: Only `POST` requests are accepted; all other methods return `errors.New("webhook must be a POST request")` immediately.
2. **Header Extraction**: `X-Whoop-Signature` header is required; missing signature → `errors.New("missing X-Whoop-Signature header")`.
3. **Bounds Protection**: `io.LimitReader(r.Body, 1<<20)` caps reads at 1MB (`maxWebhookBodySize` constant) to prevent OOM attacks.
4. **Single-Pass Hashing**: `io.TeeReader(limitedBody, mac)` feeds bytes simultaneously to the JSON decoder and the HMAC-SHA256 hasher. After decoding, remaining bytes are drained via `io.Copy(io.Discard, tee)` to ensure the full body is hashed.
5. **Signature Comparison**: The computed HMAC is `base64.StdEncoding` encoded and compared against the header value using `hmac.Equal()` (constant-time comparison to prevent timing attacks).
6. **JSON Error Deferral**: JSON decode errors are checked *after* signature validation, so even malformed-but-signed payloads get proper signature verification first.
7. **Body Close**: `r.Body` is closed via deferred `_ = r.Body.Close()`.

## 5. Concurrency Model
- **Token Bucket**: The `rateLimiter` uses `sync/atomic.Bool` for the enable/disable toggle and `golang.org/x/time/rate.Limiter` for the bucket itself. Both are safe for concurrent access.
- **URL Caching**: `CycleService`, `SleepService`, `RecoveryService`, and `WorkoutService` all use `sync.Once` to parse and cache their list endpoint URLs, preventing redundant allocations across goroutines.
- **Client Sharing**: A single `*whoop.Client` instance is designed to be shared across multiple goroutines. The `Do()` method clones the request (`req.Clone(ctx)`) to avoid mutation.
- **Jitter Source**: `math/rand/v2` is used for full jitter in `calculateBackoff()`. This package uses a per-goroutine seed since Go 1.22 and requires no explicit seeding.

## 6. External Integrations
- **WHOOP API v2**: Base URL `https://api.prod.whoop.com/developer/v2` (defined as `defaultBaseURL` constant in `client.go`).
- **OAuth Endpoints** (used only in `cmd/auth/`, NOT in the library):
  - Authorization: `https://api.prod.whoop.com/oauth/oauth2/auth`
  - Token Exchange: `https://api.prod.whoop.com/oauth/oauth2/token`
- **Dependencies**: Exactly one external dependency: `golang.org/x/time v0.14.0` for the token bucket rate limiter. Everything else is Go standard library.

## 7. Invariants & Red Lines
- **CRITICAL**: The `.whoop_token.json` file contains plaintext OAuth tokens. It is gitignored and MUST NEVER be committed.
- **CRITICAL**: The `io.LimitReader` cap of 1MB in `ParseWebhook()` MUST NOT be removed or increased without explicit security review.
- **CRITICAL**: `r.Body` MUST NOT be consumed before calling `ParseWebhook()` — the function relies on single-pass stream consumption via `TeeReader`.
- **CRITICAL**: Backoff base and max durations have defensive floors in `calculateBackoff()` (`base <= 0` defaults to 1s, `max <= 0` defaults to 60s) to prevent negative or zero-duration sleeps. These floors are NOT in the Option functions — do not add validation there without updating `calculateBackoff()`.
- **CRITICAL**: The `Client.String()` and `Client.GoString()` methods redact the OAuth token. Do not add logging that bypasses these methods to print `c.token` directly.
- **CRITICAL**: Body drains during 429 retries and error handling use `io.LimitReader(resp.Body, 4096)` to cap reads to 4KB. Do not remove this cap.

## 8. Error Handling Strategy
- Standard Go `if err != nil` propagation throughout.
- Errors are wrapped with `fmt.Errorf("context: %w", err)` for unwrapping via `errors.Is()` and `errors.As()`.
- Three custom error types provide structured error handling:
  - `*APIError`: `StatusCode int`, `Message string`, `URL string`, `Err error` (optional underlying)
  - `*RateLimitError`: `RetryAfter int` (seconds, 0 if no `Retry-After` header), `Err error` (wraps `*APIError`)
  - `*AuthError`: `StatusCode int`, `Message string`, `Err error` (wraps `*APIError`)
- All error types have `Err` typed as `error` (not `*APIError`) for interface flexibility, but `mapHTTPError()` always sets it to a `*APIError` instance.
- The `mapHTTPError()` function truncates error bodies at 1000 characters to prevent log flooding from large error responses.
- Webhook errors are plain `errors.New()` values (not typed errors) — they are simple sentinel strings.
- Transient 429 errors are handled automatically by the retry loop; only exhausted retries surface the error to the consumer.

## 9. CI/CD Pipeline
- **GitHub Actions** (`.github/workflows/ci.yml`): Runs on push to `main` and pull requests to `main`.
  - Uses `actions/checkout@v4` and `actions/setup-go@v5` with `go-version-file: go.mod`
  - `go mod tidy` → `go vet ./...` → `go test -v -race -cover ./...` → `golangci-lint` via `golangci/golangci-lint-action@v7` (v2.10.1, 5m timeout)
- **Pre-commit Hook** (`.githooks/pre-commit`): Runs `make lint` locally before every commit. Installed via `make setup`.

## 10. Local Development
- **Install & Setup**: `make setup` (configures local git hooks path to `.githooks/`, `chmod +x .githooks/*`)
- **Run Example**: `cp .env.example .env`, fill in `WHOOP_OAUTH_TOKEN` and `WHOOP_WEBHOOK_SECRET`, `source .env`, `make build-local && ./bin/example`
- **Get OAuth Token**: `export WHOOP_CLIENT_ID=... WHOOP_CLIENT_SECRET=... && go run cmd/auth/main.go`
  - First run opens a browser for the authorization flow and saves the session to `.whoop_token.json`
  - Subsequent runs automatically refresh the token using the saved `refresh_token` — no browser login needed
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

## 12. API Endpoint Map

Reference of all WHOOP API v2 paths used by the SDK:

| Service | Method | HTTP | Path |
|---------|--------|------|------|
| `CycleService` | `GetByID` | `GET` | `/cycle/{id}` |
| `CycleService` | `List` | `GET` | `/cycle` |
| `WorkoutService` | `GetByID` | `GET` | `/activity/workout/{id}` |
| `WorkoutService` | `List` | `GET` | `/activity/workout` |
| `SleepService` | `GetByID` | `GET` | `/activity/sleep/{id}` |
| `SleepService` | `List` | `GET` | `/activity/sleep` |
| `RecoveryService` | `GetByID` | `GET` | `/cycle/{cycleID}/recovery` |
| `RecoveryService` | `List` | `GET` | `/recovery` |
| `UserService` | `GetBasicProfile` | `GET` | `/user/profile/basic` |
| `UserService` | `GetBodyMeasurement` | `GET` | `/user/measurement/body` |