# whoop-go Status
Last updated: 2026-04-14

_This file tracks the detailed explore/plan/build/test sub-phases per feature. It is the single source of truth for "where am I?" Agents should update this file constantly after completing tasks or making progress, serving as the central continuity node between independent agent runs._

## Current Focus
Ready for feature ideation.

## State of Work
_List the lifecycle phases for the current SDK feature and check them off as you progress. Tailor the phases to the specific feature being built._
- [ ] Ideate & Explore: `docs/explorations/YYYY-MM-DD-{topic}.md`
- [ ] Plan (API Design & Struct Mapping): `docs/plans/YYYY-MM-DD-{feature}-api.md`
- [ ] Build (Types & Parsing: strict struct definitions mapped to WHOOP JSON)
- [ ] Build (Service Methods & Iterators)
- [ ] Build (Documentation & Godoc: ensuring 100% godoc coverage for new exported types)
- [ ] Test (Mock Server handlers + unit tests with race detector)
- [ ] Review (Memory Safety, Error Handling & Bounds Checking)
- [ ] Ship (Version Tag, CHANGELOG & GitHub Release)

## Recently Completed
_Bullet points of features or major tasks that were recently shipped. Move items here after the "Ship" phase._
- [v1.2.0 Release — Testing & Review Audit] (shipped 2026-04-20)
- [Project Bootstrap] (shipped 2026-04-14)
- [.agent/ Documentation Hardening — initial accuracy pass] (shipped 2026-04-14)
- [.agent/ Documentation Hardening v2 — comprehensive 24-issue audit with full struct field reference, API endpoint map, and corrected error type signatures] (shipped 2026-04-14)

## Known Issues
_List any persistent bugs or architectural debt that isn't blocking the current release but needs to be tracked. Do NOT list "blocks release" bugs here (those go in TESTING.md)._
- (None currently)

## What's Next
_What feature or task should be picked up after the current focus is complete? Reference the exploration doc recommendation._

## Relevant Files for Current Task
_List ONLY the files the agent needs to read or modify for the immediate next task. This prevents agents from wasting context window tokens reading the entire project._

## Review Results
_Populated during the Review phase. Keep the most recent review here; archive older ones with shipped features._

### Review Results — 2026-04-14 (v2 Audit)
- **Architecture**: pass — 12 inaccuracies corrected (Content-Type injection, body drain caps, WorkoutService sync.Once, OAuth URL scoping, CI versions, auto-refresh, cmd tests, defensive floor clarification)
- **Philosophy**: pass — 3 inaccuracies corrected (math/rand/v2, rate limiter default, ParseWebhook design rationale)
- **Style Guide**: pass — 4 inaccuracies corrected (RetryAfter type, AuthError.Err type, Content-Type injection, test descriptions)
- **Testing**: pass — 3 inaccuracies corrected (missing cmd test, wrong test description, auto-refresh)
- **Status**: pass — updated records
- **Complete struct field reference added**: All 16 domain types fully documented field-by-field
- **API endpoint map added**: All 10 REST endpoints catalogued

### Review Results - 2026-04-20 (Release Readiness)
- **Architecture**: pass — The codebase rigorously follows the `ARCHITECTURE.md` patterns. Sub-services use `sync.Once` URL caching, Functional Options configure the `Client` without breaking defensive limits, and the single-package layout remains intact.
- **Security**: pass — Evaluated all ingress points. Webhook payloads are properly capped at 1MB, HMAC signatures use constant-time comparisons, response body drains are capped at 4KB, and structural tokens are redacted in system logs. The zero-external-dependency posture is strictly maintained.
- **Product fit**: pass — It effectively solves the pain points by cleanly abstracting WHOOP API pagination, 429 backoff calculations, and schema validations into type-safe idiomatic Go.

### Action Items
_For each item, specify severity and routing._

| Item | Severity | Route To | Status |
|------|----------|----------|--------|
| _(None)_ | | | |

## Active Worktrees
_Track parallel agent work when using git worktrees. Remove entries during Ship phase cleanup._

| Worktree | Branch | Status | Owner |
|----------|--------|--------|-------|
| (none — sequential execution) | | | |

---

## Stub Audit Tracker

N/A — Not a full-stack project. No frontend stubs.

---

## Prompt Versioning Changelog

N/A — No LLM prompts in this project.