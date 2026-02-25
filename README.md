# whoop-go

[![Go Reference](https://pkg.go.dev/badge/github.com/arvarik/whoop-go/whoop.svg)](https://pkg.go.dev/github.com/arvarik/whoop-go/whoop)
[![CI](https://github.com/arvarik/whoop-go/actions/workflows/ci.yml/badge.svg)](https://github.com/arvarik/whoop-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/arvarik/whoop-go)](https://goreportcard.com/report/github.com/arvarik/whoop-go)

`whoop-go` is a production-grade, highly robust Go client library for integrating with the WHOOP API.

Engineered with resiliency in mind, it natively handles strict WHOOP rate limits (100 req/min) through transparent token buckets and exponential backoff jitter. It securely extracts HTTP webhooks using HMAC-SHA256 validations, and surfaces strongly-typed structs mapping the complex WHOOP domain (Cycles, Workouts, Sleep, and Recovery) via seamless pagination iterators.

**Disclaimer:** This is an unofficial open-source library and is not affiliated with, endorsed by, or supported by WHOOP.

## Features

- **Built-in Resilience**: Implements an intrinsic thread-safe token bucket rate-limiter enforcing the 100 req/min and 10,000 req/day WHOOP API quotas. Automatically intercepts HTTP `429 Too Many Requests` responses, sleeping utilizing randomized exponential backoffs before retrying safely.
- **Webhook Verifier**: Features `whoop.ParseWebhook(r, secret)`, dynamically digesting inbound HTTP requests, safely streaming payloads, validating `X-Whoop-Signature` HMAC-SHA256 authenticity hashes without memory leaks, and returning structured skinny webhook types (`workout.updated`, `cycle.updated`, etc.).
- **Iterator Pagination**: Converts cumbersome `next_token` URL query cursor traversals into a deeply idiomatic Go iterator pattern utilizing `.NextPage(ctx)`.
- **Zero External Dependencies**: Outside of the foundational Golang `golang.org/x/time/rate` token bucket algorithm, the client is strictly built upon Go standard primitives (`net/http`, `crypto/hmac`).

## Component Architecture

The library is modularized by functional domains to provide strict operational boundaries:

- `client.go`: Centralizes the configurable HTTP `Client`, injects global functional options (`WithToken`, `WithBackoffBase`), and integrates the `rateLimiter`.
- `ratelimit.go`: Enforces concurrency-safe request throttling.
- `webhooks.go`: Provides high-performance, single-pass webhook stream consumption and signature validation.
- `profile.go`: Manages athlete basic profiles and highly granular `BodyMeasurement` data.
- `cycle.go`: Maps the overarching physiological day with calculated `Strain` and `HeartRate` arrays.
- `sleep.go` & `recovery.go`: Exposes complex sleep staging, restorative data markers, and systemic nervous recovery scorings.
- `workout.go`: Maps chronological activity zone durations and absolute metric loads.

## Installation

You need Go `1.22` or higher installed.

```bash
go get github.com/arvarik/whoop-go/whoop
```

## Quick Start: The Webhook Example

The repository includes a fully-functional Webhook + REST architecture example at [`cmd/example/main.go`](cmd/example/main.go). This script demonstrates how to securely parse live webhooks and trigger a secondary data scrape pipeline instantly.

### Running the Example
```bash
# Copy the example .env and fill in your credentials
cp .env.example .env
vim .env   # Add your WHOOP_OAUTH_TOKEN and WHOOP_WEBHOOK_SECRET
source .env

# Start the Webhook Listener on Port 8080
make build-local
./bin/example
```

### Verifying Credentials with cURL

Before building an integration, you can validate your OAuth token directly against the WHOOP API:

```bash
# Set your token
export WHOOP_OAUTH_TOKEN="your_oauth2_token_here"

# 1. Fetch your basic profile
curl -s -H "Authorization: Bearer $WHOOP_OAUTH_TOKEN" \
  "https://api.prod.whoop.com/developer/v1/user/profile/basic" | jq

# 2. Fetch your body measurements
curl -s -H "Authorization: Bearer $WHOOP_OAUTH_TOKEN" \
  "https://api.prod.whoop.com/developer/v1/user/measurement/body" | jq

# 3. List recent physiological cycles (last 10)
curl -s -H "Authorization: Bearer $WHOOP_OAUTH_TOKEN" \
  "https://api.prod.whoop.com/developer/v1/cycle?limit=10" | jq

# 4. List recent workouts
curl -s -H "Authorization: Bearer $WHOOP_OAUTH_TOKEN" \
  "https://api.prod.whoop.com/developer/v1/activity/workout?limit=10" | jq

# 5. List recent sleep events
curl -s -H "Authorization: Bearer $WHOOP_OAUTH_TOKEN" \
  "https://api.prod.whoop.com/developer/v1/activity/sleep?limit=10" | jq

# 6. List recent recovery scores
curl -s -H "Authorization: Bearer $WHOOP_OAUTH_TOKEN" \
  "https://api.prod.whoop.com/developer/v1/recovery?limit=10" | jq
```

### Under the Hood
1. A WHOOP event triggers your `webhook.updated` skinny payload to exactly `:8080/whoop/webhook`.
2. The `whoop.ParseWebhook()` helper validates the signature and asserts the `event.Type`.
3. An explicit asynchronous `processWorkout()` GoRoutine utilizes `client.Workout.GetByID(ctx, event.ID)` to query the full `Workout` schema for local logging and storage!

## Example API Calls

The library provides individual services attached to the core `Client`. Below are comprehensive examples showing basic interactions.

### 1. Initializing the Client

Configuration is heavily customizable utilizing the Functional Options pattern.

```go
import "github.com/arvarik/whoop-go/whoop"

client := whoop.NewClient(
    whoop.WithToken("your_oauth_token"),
    whoop.WithMaxRetries(5),               // Automatically retry failures or 429 limits up to 5 times
    whoop.WithBaseURL("https://custom.proxy.example.com"), // Optional: override base URL
)
```

### 2. Validating Webhooks

```go
http.HandleFunc("/whoop/webhook", func(w http.ResponseWriter, r *http.Request) {
    event, err := whoop.ParseWebhook(r, "my_webhook_secret_key")
    if err != nil {
        log.Printf("Fraudulent webhook prevented: %v", err)
        w.WriteHeader(http.StatusUnauthorized)
        return
    }

    log.Printf("Received genuine webhook event! Type: %s, ID: %d", event.Type, event.ID)
})
```

### 3. Fetching Cycles via Iterator Pagination

The WHOOP API caps returns at 50 arrays. You sequentially traverse all history easily via the `NextPage(ctx)` cursor logic.

```go
// Fetch the initial block of Cycles
page, err := client.Cycle.List(ctx, &whoop.ListOptions{Limit: 10})
if err != nil {
    log.Fatal(err)
}

for {
    // Process current batch
    for _, cycle := range page.Records {
        fmt.Printf("Cycle: %d, Strain: %.1f\n", cycle.ID, cycle.Score.Strain)
    }

    // Traverse pagination seamlessly!
    page, err = page.NextPage(ctx)
    if err != nil {
        if errors.Is(err, whoop.ErrNoNextPage) {
            break // Done!
        }
        log.Fatal(err)
    }
}
```

## Local Development / First Time Setup

If you are contributing to this library, you should run the `setup` command immediately after cloning. This automatically configures standard Git hooks to invoke the Go linter before allowing commits:

```bash
# Sets `core.hooksPath` to `.githooks` enabling the pre-commit action automatically
make setup
```

Run test coverage utilizing mocked REST architectures safely disconnected from Live APIs:

```bash
make test
```

## License

This project is licensed under the MIT License.
