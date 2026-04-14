# Product Philosophy

_This is the soul of the product. It explains why the app exists and what its core beliefs are. Product Visionaries and UI/UX Designers use this to make feature and design decisions. Engineers use it to resolve ambiguity._

## 1. Why This Exists
Integrating with rate-limited, domain-heavy APIs is error-prone. This project exists to provide a production-grade, highly robust Go client library for integrating with the WHOOP API v2. It removes the boilerplate of handling strict quotas, pagination, and webhook validation, allowing developers to focus on analyzing the data.

## 2. Target User
Go developers building resilient applications, data pipelines, or health dashboards that integrate with WHOOP user data. They require reliability, type safety, and efficient background processing.

## 3. Core Beliefs
- **Built-in Resilience**: The client must natively handle strict WHOOP rate limits (100 req/min) through transparent token buckets and exponential backoff jitter. Developers shouldn't have to manually sleep threads for HTTP 429s.
- **Zero External Dependencies**: Beyond the foundational `golang.org/x/time/rate` token bucket, the library must rely strictly on Go standard primitives (`net/http`, `crypto/hmac`). Lean and stable.
- **Secure by Default**: Webhooks must be easily validated via single-pass HMAC-SHA256 authenticity hashes without risking memory leaks from oversized payloads.
- **Idiomatic Go**: Complex domain traversals (like pagination) must be simplified using idiomatic Go patterns (e.g., `.NextPage(ctx)` iterator).

## 4. Design & UX Principles
- **Consumer-Friendly API**: The flat `whoop/` package layout and Functional Options configuration pattern provide an ergonomic and instantly recognizable developer experience.
- **Fail Fast, Fail Safely**: When an invariant is broken (e.g., a missing signature or oversized payload), the library must return a clear, typed error immediately to prevent system degradation.

## 5. What This Is NOT
- This is NOT an official WHOOP library.
- This is NOT a frontend framework or a visual dashboard. It is strictly a backend HTTP client and domain mapper.
- This is NOT a generic wrapper for any REST API. It is hyper-specialized for WHOOP API v2 schemas and constraints.