# Product Philosophy

_This is the soul of the product. It explains why the app exists and what its core beliefs are. Engineers use it to resolve ambiguity when designing systems and structuring code._

## 1. Why This Exists
Integrating with rate-limited, domain-heavy APIs is error-prone, repetitive, and riddled with subtle bugs around pagination tokens, retry storms, and credential leaks. This project exists to provide a production-grade Go client library for the WHOOP API v2 that absorbs all of that complexity—developers write business logic, not HTTP plumbing.

## 2. Target User
Go developers building resilient applications, data pipelines, health dashboards, or webhook-driven microservices that integrate with WHOOP user data. They expect reliability, compile-time type safety, clean concurrency patterns, and zero surprises from the underlying transport layer.

## 3. Core Beliefs

### Zero External Dependencies
Beyond the foundational `golang.org/x/time/rate` token bucket algorithm, the library relies strictly on Go standard library primitives (`net/http`, `crypto/hmac`, `encoding/json`, `io`, `sync/atomic`, `math/rand/v2`). We do not pull in third-party HTTP clients (`go-resty`, `req`), configuration frameworks (`viper`), or logging libraries (`zap`, `logrus`). Fewer dependencies means fewer CVEs, fewer version conflicts, and faster compilation.

### Built-in Hardware Resilience
The client natively handles WHOOP's strict rate limits (100 req/min) through a transparent token bucket and exponential backoff with full jitter (computed via `math/rand/v2`). Consumers should not need to implement their own retry logic or manually sleep on HTTP 429 responses—the SDK absorbs rate-limit turbulence internally and only surfaces errors after retries are exhausted.

### Strict Structural Determinism
We reject `interface{}`/`any` for data modeling. Every WHOOP API response is mapped to a concrete Go struct with precise JSON tags matching the upstream schema. This provides compile-time type checking and enables IDE autocompletion. The only use of `any` is the type parameter constraint in the generic `paginatedResponse[T any]` type, which is a standard Go generics pattern—not a dynamically typed container.

### Secure by Default
Webhooks are hostile ingress vectors. The library enforces:
- **Payload size caps** via `io.LimitReader` (1MB) to prevent OOM attacks.
- **Single-pass HMAC-SHA256** via `io.TeeReader` for efficient, zero-copy signature verification.
- **Constant-time comparison** via `hmac.Equal()` to prevent timing side-channel attacks.
- **Token redaction** in `String()` / `GoString()` to prevent credential leaks in logs.
- **Body drain caps** of 4KB during retry loops and error handling to prevent memory exhaustion from large error bodies.

### Idiomatic Go
Complex domain traversals (like pagination) use idiomatic Go patterns—iterator methods like `.NextPage(ctx)` and sentinel errors like `ErrNoNextPage` for `errors.Is()` checks. Error types implement `Unwrap()` for the standard `errors` ecosystem. Context cancellation is honored at every blocking point (rate limiter wait, backoff sleep, HTTP transport).

## 4. Design & UX Principles

### Consumer-Friendly Bootstrapping
The flat `whoop/` package layout and Functional Options pattern (`whoop.NewClient(whoop.WithToken("..."))`) provide an ergonomic and instantly recognizable developer experience. Sensible defaults (3 retries, 1s base backoff, 60s max backoff, 30s HTTP timeout, rate limiter enabled at 100 req/min with burst of 100) mean the zero-config path just works.

### Sub-Service Organization
Domain resources are accessed via named sub-services on the client: `client.User`, `client.Cycle`, `client.Sleep`, `client.Workout`, `client.Recovery`. This mirrors the WHOOP API structure and makes discoverability trivial via IDE autocompletion.

Webhook parsing is intentionally a **package-level function** (`whoop.ParseWebhook(r, secret)`) rather than a service method, because it operates on raw HTTP requests independently of the `Client` instance and its authentication/rate-limiting machinery.

### Fail Fast, Fail Safely
When an invariant is broken (missing webhook signature header, oversized payload, invalid config, exhausted retries), the library returns a clear, strongly-typed error immediately. The library does NOT log internally—all diagnostic information flows through returned errors, giving the consumer full control over observability.

## 5. What This Is NOT
- This is NOT an official WHOOP library. It is a community-maintained open-source project.
- This is NOT a frontend framework or a visual dashboard. It is strictly a backend HTTP client and domain mapper.
- This is NOT a generic wrapper for any REST API. It is hyper-specialized for WHOOP API v2 schemas, endpoints, and rate-limit constraints.
- This is NOT a full OAuth 2.0 library. The `cmd/auth/` helper is a convenience tool for local development, not a production-grade token management solution.
- This is NOT a logging library. The SDK contains zero `log.Printf()` calls — diagnostics flow exclusively through returned errors.