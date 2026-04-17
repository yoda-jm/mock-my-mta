# Improvement Plan — Status Tracker

## Phase 1 — Critical fixes

| # | Task | Status |
|---|------|--------|
| 1 | Graceful shutdown with signal handling | DONE |
| 2 | Remove panic-restart loop | DONE |
| 3 | Fix README SMTP port (1025 not 8025) | DONE |
| 4 | Health check verifies SMTP port | DONE |
| 5 | Quote attachment filenames | DONE |
| 6 | Proper HTML sanitization (bluemonday) | DONE |
| 7 | Gate pprof behind --debug flag | DONE |

## Phase 2 — Storage and performance

| # | Task | Status |
|---|------|--------|
| 8 | SQLite storage with indexes | DONE |
| 9 | Email count caching | DONE |
| 10 | In-memory storage | DONE |
| 11 | Clean up config (scope-based routing) | DONE |
| 12 | File locking for concurrent writes | PENDING |

## Phase 3 — SMTP protocol

| # | Task | Status |
|---|------|--------|
| 13 | STARTTLS with self-signed cert | DONE |
| 14 | Inbound SMTP AUTH | DONE |
| 15 | Configurable message SIZE limit | DONE |
| 16 | Bounce/DSN simulation | PENDING |
| 17 | Configurable failure injection | PENDING |

## Phase 4 — Real-time and API

| # | Task | Status |
|---|------|--------|
| 18 | WebSocket real-time notifications | DONE |
| 19 | GET /api/health | DONE |
| 20 | GET /api/stats | DONE |
| 21 | Wait-for-email API | DONE |
| 22 | OpenAPI/Swagger spec | PENDING |
| 23 | Environment variable configuration | PENDING |

## Phase 5 — Frontend UX

| # | Task | Status |
|---|------|--------|
| 24 | Compose/send test email from UI | PENDING |
| 25 | Keyboard shortcuts | PENDING |
| 26 | Read/unread tracking | PENDING |
| 27 | Body content search (was broken) | DONE |
| 28 | Header-based search | PENDING |
| 29 | Search export (JSON/CSV) | PENDING |
| 30 | Email threading | PENDING |

## Phase 6 — Integration and deployment

| # | Task | Status |
|---|------|--------|
| 31 | Docker Compose with TLS | PENDING |
| 32 | Persistent volume mount | PENDING |
| 33 | Webhook notifications | PENDING |
| 34 | API documentation in README | PENDING |
| 35 | Configuration documentation | PENDING |

## Bonus (not in original plan)

| Feature | Status |
|---------|--------|
| Dark mode with CSS custom properties | DONE |
| UI redesign (branded sidebar, nav tabs, compact list) | DONE |
| Bulk select/delete/relay with confirmation | DONE |
| .eml download, raw headers, MIME tree with actions | DONE |
| MIME preview modal | DONE |
| E2E screenshot capture (55 tests) | DONE |
| GitHub Pages Playwright report | DONE |
| Scoped storage engine (per-operation routing) | DONE |
| Parse-once optimization (setWithID takes []byte) | DONE |
| Storage layer design document | DONE |
| Hash-based URL routing for deep linking | DONE |
| gotoEmail/gotoSearch page model methods | DONE |
| waitForEmail returns URL + total_matches | DONE |
| watch-html body fix in multipart/alternative | DONE |
| log.Fatalf removed from matcher (no more crashes) | DONE |
| 90+ Go unit tests | DONE |

## Summary

**Done: 25/35 original items + 16 bonus = 41 total features**

## Remaining by priority

| # | Task | Effort | Impact |
|---|------|--------|--------|
| 25 | Keyboard shortcuts (j/k, d, r, /) | Low | High — power users |
| 23 | Environment variable configuration | Medium | High — Docker/K8s |
| 24 | Compose/send test email from UI | Medium | High — no external SMTP client needed |
| 17 | Configurable failure injection | Medium | High — chaos testing |
| 34 | API documentation in README | Medium | Medium — developer onboarding |
| 28 | Header-based search | Low | Medium — debugging custom headers |
| 29 | Search export (JSON/CSV) | Low | Medium — CI reporting |
| 26 | Read/unread tracking | Low | Low |
| 12 | File locking | Medium | Low — dev tool, single writer |
| 16 | Bounce/DSN simulation | Medium | Low — niche |
| 22 | OpenAPI spec | Medium | Low |
| 30 | Email threading | High | Low |
| 31-35 | Docker TLS, volumes, webhooks, docs | Low-Med | Low |
