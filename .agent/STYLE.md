# Style Guide & Code Conventions

_This document enforces the coding patterns and structural rules of the project. It prevents drift as multiple agents work on the codebase. Agents MUST follow these rules strictly._

## 1. Visual Language & Tokens
- N/A — Go SDK library project with no frontend UI.

## 2. Component Patterns
- N/A — Go SDK library project with no frontend UI.

## 3. Code Conventions

### Architecture Patterns
- **Library Layout**: Flat structure within `whoop/` — no sub-packages. All source files coexist at the top level to ensure simple consumer imports (`github.com/arvarik/whoop-go/whoop`). Executable examples and tools reside exclusively in `cmd/`.
- **Functional Options**: Configuration for the `Client` is done exclusively using the Functional Options pattern (e.g., `WithToken()`, `WithMaxRetries()`, `WithBaseURL()`). Never expose `Client` struct fields as public; all configuration flows through `Option` functions.
- **Sub-Service Pattern**: Domain resources are organized as named services on the `Client` struct (`client.User`, `client.Cycle`, `client.Sleep`, `client.Workout`, `client.Recovery`). Each service holds a back-reference to the parent `Client` for accessing `Do()`.
- **Iterator Pagination**: Pagination is abstracted into typed `*XxxPage` structs with a `NextPage(ctx)` method. Consumers iterate until `errors.Is(err, whoop.ErrNoNextPage)`. Never expose raw cursor tokens in the public API surface.
- **URL Caching**: Services with `List()` methods may use `sync.Once` to parse and cache their endpoint URLs, preventing redundant `url.Parse()` calls across goroutines.

### Struct & Type Conventions
- **JSON Tag Alignment**: Struct field order MUST match the upstream WHOOP API JSON schema for readability. Do not reorder fields for alignment optimization (the `fieldalignment` linter is explicitly disabled for this reason).
- **Pointer Fields for Optionals**: Use pointer types (`*float64`, `*int`, `*time.Time`, `*ScoreType`) for fields that may be `null` or omitted in the WHOOP API response. This distinguishes "absent" from "zero value".
- **Embedded Score Types**: Scores (e.g., `*WorkoutScore`, `*SleepScore`, `*RecoveryScore`) are always optional pointer fields tagged `json:"score,omitempty"` because the API returns `null` when `score_state` is not `"SCORED"`.

### Error Handling
- **Wrapping**: Use `fmt.Errorf("context: %w", err)` to wrap errors for the `errors.Is()`/`errors.As()` ecosystem.
- **Typed Errors**: The SDK defines three error types:
  - `*APIError`: Generic HTTP errors (4xx/5xx) with `StatusCode`, `Message`, `URL`, and optional underlying `Err`.
  - `*RateLimitError`: HTTP 429 errors with `RetryAfter` duration and underlying `*APIError`.
  - `*AuthError`: HTTP 401/403 errors with `StatusCode`, `Message`, and underlying `*APIError`.
- All error types implement `Unwrap() error` for chain inspection.
- **Body Truncation**: Error response bodies are truncated to 1000 characters in `mapHTTPError()` to prevent log flooding.
- **No Internal Logging**: The library does NOT use `log.Printf()` or any logging framework. All diagnostic information flows through returned errors.

### Thread Safety
- The `*Client` struct is designed for concurrent use by multiple goroutines.
- `Do()` clones the request via `req.Clone(ctx)` to prevent mutation of shared request objects.
- The rate limiter uses `sync/atomic.Bool` for its enable/disable toggle.
- Services using `sync.Once` for URL caching are safe for concurrent `List()` calls.

## 4. Naming Conventions
- **Files**: `snake_case.go` (e.g., `client.go`, `webhook_test.go`, `mock_server_test.go`).
- **Variables / Functions**: Idiomatic Go—`camelCase` for unexported, `PascalCase` for exported.
- **JSON Tags**: Must precisely map to the WHOOP API v2 `snake_case` schema (e.g., `json:"user_id"`, `json:"score_state"`).
- **Test Files**: `*_test.go` suffix. Test helpers use `t.Helper()`. Mock infrastructure lives in `mock_server_test.go`.

## 5. Import Ordering
Enforced by `goimports` (configured in `.golangci.yml`):
1. Standard library (`context`, `encoding/json`, `fmt`, `net/http`, `sync`)
2. Third-party (`golang.org/x/time/rate`)
3. Internal packages (`github.com/arvarik/whoop-go/whoop` — used only in `cmd/`)

## 6. Linting Configuration
- **Tool**: `golangci-lint` v2 (config in `.golangci.yml`).
- **Enabled linters**: `errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`.
- **Formatters**: `gofmt`, `goimports`.
- **Disabled `govet` analyzers**:
  - `fieldalignment`: Struct field order matches the JSON API schema for readability.
  - `shadow`: Too noisy for idiomatic Go error handling patterns (re-using `err` in nested scopes).
- **`errcheck` config**: `check-blank: false` — allows `_ = resp.Body.Close()` patterns.
- **Timeout**: 5 minutes for CI runs.
- **RULE**: Do NOT suppress linter warnings via inline `//nolint:` comments. If a rule needs adjustment, modify `.golangci.yml`.

## 7. Testing Strategy
- **Mock Server, Not Mock Interfaces**: Tests use `net/http/httptest.Server` (built in `mock_server_test.go`) to spin up an ephemeral HTTP multiplexer with literal JSON payloads. The real `Client` code is tested against the fake network via `whoop.WithBaseURL(ts.URL)`.
- **`newMockServer(t)`**: Creates the shared test server with route handlers for all domains + error scenarios (429, 403, context cancellation delay).
- **`newMockClient(ts, ...opts)`**: Builds a `Client` pointed at the mock server with shorter backoff settings so tests don't stall.
- **Race Detector**: All tests run with `-race` flag (`make test`). This is non-negotiable.
- **Test Package**: Tests live in `package whoop` (not `whoop_test`) to access unexported internals. Example tests (`example_test.go`) use the external test package for godoc examples.

## 8. Documentation Standards
- **godoc**: All exported types, interfaces, constants, and functions MUST have godoc-compatible comments (`// FunctionName does X.`). See `doc.go` for package-level documentation with runnable examples.
- **README.md**: Focused strictly on installation, quick start, and usage examples. Architectural details belong in `.agent/ARCHITECTURE.md`.

## 9. Anti-Patterns (FORBIDDEN)
- ❌ NEVER use `interface{}` / `any` as a data container. The WHOOP domain is strongly-typed. (Exception: `any` as a type parameter constraint in generics like `paginatedResponse[T any]` is acceptable—that's standard Go generics.)
- ❌ NEVER shadow errors or silently discard them without explicit justification.
- ❌ NEVER commit `.whoop_token.json`, `.env`, or plaintext OAuth tokens to version control.
- ❌ NEVER disable linter checks via inline `//nolint:` comments. Use `.golangci.yml` configuration.
- ❌ NEVER consume the HTTP request `r.Body` before passing it to `whoop.ParseWebhook()`. This destroys the webhook signature validation stream.
- ❌ NEVER add logging (`log.Printf`, `fmt.Println`) to the library code. Diagnostics flow through returned errors.
- ❌ NEVER add external dependencies without explicit justification and security review. The zero-dependency mandate is a core design belief.
- ❌ NEVER reorder struct fields for memory alignment if it breaks visual correspondence with the WHOOP API JSON schema.