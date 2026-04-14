# Style Guide & Code Conventions

_This document enforces the visual identity and coding patterns of the project. It prevents context drift as multiple agents work on the codebase. Agents MUST follow these rules strictly._

## 1. Visual Language & Tokens
- N/A — Go SDK library project with no frontend UI.

## 2. Component Patterns
- N/A — Go SDK library project with no frontend UI.

## 3. Code Conventions

### Architecture Patterns
- **Library Layout**: Flat structure within the `whoop/` directory for the core package to ensure consumer-friendly imports (`github.com/arvarik/whoop-go/whoop`). Executable examples reside in `cmd/`.
- **Functional Options**: Configuration for the `Client` is done exclusively using the Functional Options pattern (e.g., `WithToken()`, `WithMaxRetries()`).
- **Iterator Pagination**: API pagination logic is abstracted into a seamless iterator pattern (`page.NextPage(ctx)`), preventing consumers from manually managing cursors.
- **Thread Safety**: Core components like the rate limiter and the configured HTTP client must remain safe for concurrent use by multiple goroutines.

### Strict Typing & Linting
- All code MUST pass `golangci-lint run ./...` with zero warnings.
- The `govet` shadow rule is explicitly disabled to avoid excessive noise for idiomatic Go error handling.
- The `govet` fieldalignment rule is disabled to ensure Go struct field ordering closely matches the WHOOP JSON API schema for optimal readability.

## 4. Naming Conventions
- **Files**: `snake_case.go` (e.g., `client.go`, `webhook_test.go`).
- **Variables / Functions**: Idiomatic Go `camelCase` for unexported package internals, `PascalCase` for exported types and functions.
- **JSON Tags**: Must precisely map to the WHOOP API v2 `snake_case` schema.

## 5. Import Ordering
- Enforced by `goimports`:
  1. Standard library (`fmt`, `net/http`, `context`)
  2. Third-party (`golang.org/x/time/rate`)
  3. Internal packages (not applicable for the flat `whoop/` structure, but applies to `cmd/`)

## 6. Documentation Standards
- **godoc**: All exported types, interfaces, constants, and functions MUST have godoc-compatible comments (`// FunctionName does X.`).
- Keep the `README.md` focused strictly on setup, quick start, and usage examples. Architectural details belong in `.agent/ARCHITECTURE.md`.

## 7. Anti-Patterns (FORBIDDEN)
- ❌ NEVER use `interface{}` / `any` where a concrete type or struct exists. The WHOOP domain is strongly-typed.
- ❌ NEVER shadow errors or ignore them without explicit justification or handling (unless configuring linter overrides).
- ❌ NEVER commit `.whoop_token.json` or plaintext OAuth tokens to version control.
- ❌ NEVER disable `golangci-lint` checks via inline comments unless it's an extreme edge case (use the `.golangci.yml` configuration instead).
- ❌ NEVER consume the HTTP request `r.Body` inside an application handler before passing it to `whoop.ParseWebhook()`, as this will destroy the webhook signature validation stream.