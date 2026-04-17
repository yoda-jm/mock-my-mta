# Improvement Plan — Status Tracker

## Phase 1 — Critical fixes

| # | Task | Status | Notes |
|---|------|--------|-------|
| 1 | Graceful shutdown with signal handling | DONE | 5s timeout context, SIGQUIT/SIGTERM/SIGINT |
| 2 | Remove panic-restart loop | DONE | Servers fail fast, let Docker restart |
| 3 | Fix README SMTP port (1025 not 8025) | DONE | |
| 4 | Health check verifies SMTP port | DONE | nc -z localhost 1025 in Dockerfile |
| 5 | Quote attachment filenames in Content-Disposition | DONE | fmt.Sprintf with quotes |
| 6 | Proper HTML sanitization (bluemonday) | DONE | microcosm-cc/bluemonday UGCPolicy |
| 7 | Gate pprof behind --debug flag | DONE | |

## Phase 2 — Storage and performance

| # | Task | Status | Notes |
|---|------|--------|-------|
| 8 | Implement SQLite storage with indexes | DONE | modernc.org/sqlite (pure Go, no CGo) |
| 9 | Email count caching (avoid full scan) | DONE | Memory layer caches parsed headers |
| 10 | Implement in-memory storage | DONE | Parsed cache with sync.RWMutex |
| 11 | Clean up config (remove misleading backends) | DONE | Scope-based routing (read/search/write/raw/cache/all) |
| 12 | File locking for concurrent SMTP writes | PENDING | |

## Phase 3 — SMTP protocol

| # | Task | Status | Notes |
|---|------|--------|-------|
| 13 | STARTTLS with self-signed cert | DONE | Auto-generated ECDSA cert in memory |
| 14 | Inbound SMTP AUTH (accept any credentials) | DONE | PLAIN + LOGIN via chrj/smtpd |
| 15 | Configurable message SIZE limit | PENDING | |
| 16 | Bounce/DSN simulation mode | PENDING | |
| 17 | Configurable failure injection (chaos testing) | PENDING | |

## Phase 4 — Real-time and API

| # | Task | Status | Notes |
|---|------|--------|-------|
| 18 | WebSocket real-time notifications | DONE | gorilla/websocket, broadcasts new/delete events |
| 19 | GET /api/health endpoint | DONE | Checks storage accessibility |
| 20 | GET /api/stats endpoint | PENDING | |
| 21 | Wait-for-email API | PENDING | |
| 22 | OpenAPI/Swagger spec | PENDING | |
| 23 | Environment variable configuration | PENDING | |

## Phase 5 — Frontend UX

| # | Task | Status | Notes |
|---|------|--------|-------|
| 24 | Compose/send test email from UI | PENDING | |
| 25 | Keyboard shortcuts (j/k, d, r) | PENDING | |
| 26 | Read/unread email tracking | PENDING | |
| 27 | Body content search (was broken) | DONE | Case-insensitive PlainTextMatch, log.Fatalf removed |
| 28 | Header-based search (X-Custom, Reply-To) | PENDING | |
| 29 | Search results export (JSON/CSV) | PENDING | |
| 30 | Email threading / conversation grouping | PENDING | |

## Phase 6 — Integration and deployment

| # | Task | Status | Notes |
|---|------|--------|-------|
| 31 | Docker Compose with TLS certs | PENDING | |
| 32 | Persistent volume mount for email data | PENDING | |
| 33 | Webhook notifications on new email | PENDING | |
| 34 | API documentation in README | PENDING | |
| 35 | Configuration documentation | PENDING | |

## Summary

**Done: 19/35** (54%)

**Completed features not in original plan:**
- Dark mode with CSS custom properties
- UI redesign (branded sidebar, nav tabs, compact list)
- Bulk select/delete/relay with confirmation dialogs
- .eml download, raw headers view, MIME tree with actions
- MIME preview modal
- E2E screenshot capture (45 tests)
- GitHub Pages Playwright report
- Scoped storage engine (per-operation routing)
- Parse-once optimization (setWithID takes []byte)
- Storage layer design document
- 80+ Go unit tests, 45 e2e tests

## Known Limitations

- `message/rfc822` parts (forwarded emails) treated as binary blobs
- `multipart/digest` not specially handled
- No concurrent file locking (filesystem storage)
- Date-dependent e2e test (`newer_than:30d`) will need updating
- Polling fallback if WebSocket connection fails
- Page size hardcoded to 20
- Date locale hardcoded to `'fr'` in `script.js`
- SQLite search with matchers still loads raw + parses (no SQL WHERE translation yet)
