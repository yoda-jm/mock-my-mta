# Improvement Plan

Comprehensive audit of mock-my-mta — a local SMTP mock server for testing
email-sending software without spamming real addresses.

**Overall scores** (current state):

| Area | Score | Verdict |
|------|-------|---------|
| Architecture | 3/10 | No graceful shutdown, panic restarts, no context timeouts |
| SMTP protocol | 2/10 | No TLS, no AUTH, no size limits, no bounce/DSN |
| Storage | 2/10 | Filesystem only; Memory/SQLite stubbed; no indexing |
| API | 7/10 | Good endpoints, missing health check, no WebSocket |
| Email parsing | 7/10 | Good RFC support, charset done, some edge cases remain |
| Frontend UX | 7/10 | Dark mode, tabs, bulk ops, but no real-time, no compose |
| Testing | 7/10 | 45 e2e + 80 unit tests; error paths covered; some gaps |
| Security | 4/10 | XSS regex-only, pprof exposed, no rate limiting |
| Docker | 6/10 | Works but health check incomplete, no volume, port confusion |
| Documentation | 4/10 | Installation OK, API docs missing, SMTP port wrong in README |
| Performance | 3/10 | Linear search, no indexing, breaks at 10K+ emails |

---

## 1. Completed Work

### 1.1 Bugs fixed

| Bug | Fix |
|-----|-----|
| HTML-only emails show raw tags in preview | `stripHTMLTags()` in `GetPreview()` |
| No Content-Type header → empty body | Default `text/plain` injected during parsing |
| ISO-8859-1 charset → mojibake | `golang.org/x/text` charset conversion |
| RFC 2231 filename not decoded | `mime.ParseMediaType` in `GetFilename()` |
| `Paginnation` typo in API JSON | Renamed to `Pagination` |
| `phytisalLayer` / `recipents` / `GetRaEmail` typos | Fixed |
| Bootstrap 4→5 mismatch | `data-toggle` → `data-bs-toggle` |
| ~9 undeclared JS global variables | Added `const`/`let` |
| Dead CSS, commented-out code, deprecated HTML | Cleaned up |
| 1x1px invisible test images | Replaced with 40x40px colored PNGs |
| Long sender addresses overflow email list | `text-overflow: ellipsis` |

### 1.2 Features added

| Feature | Details |
|---------|---------|
| Raw headers view | `GET /api/emails/{id}/headers` + tab |
| .eml download | `GET /api/emails/{id}/download` + button |
| MIME structure tree | `GET /api/emails/{id}/mime-tree` + interactive tab |
| MIME preview modal | CID images/text previewed in modal |
| Proper Bootstrap nav-tabs | Icons for all body versions |
| External images toggle | Only shown for HTML body tabs |
| Dark mode | Full CSS custom property theming, persisted |
| UI redesign | Branded sidebar, card-style header, compact list |
| Bulk select/delete/relay | Checkboxes, select-all, confirmation dialogs |
| Email polling | 5s interval, toast on new emails |
| XSS sanitization | Server-side `<script>` and `on*` attribute stripping |
| Error path tests | 22 HTTP handler unit tests for 404/400/500 |
| E2E screenshot capture | All 45 tests attach screenshots to report |
| GitHub Pages report | Playwright report deployed on push to main |
| GitHub Actions CI | Dedicated e2e workflow + build workflow |

### 1.3 Test coverage

- **45 e2e tests** across 3 spec files
- **80+ Go unit tests** across 9 test files
- **17 API endpoints** — all covered by e2e, 60%+ by unit tests

---

## 2. Critical Bugs (fix now)

### 2.1 Graceful shutdown missing

**`cmd/server/main.go:90`** — Signal handler receives QUIT/TERM but does nothing.
Servers never stop; process must be killed.

**`cmd/server/main.go:135-149`** — Panic-restart loop masks real crashes with
1-second sleep. Should fail fast and let Docker/systemd restart.

**`http/httpd.go:193`** — `Shutdown()` uses `context.TODO()` instead of
`context.WithTimeout()`.

**Fix:** Implement proper signal handling → call `server.Shutdown(ctx)` with
5s timeout → exit cleanly.

### 2.2 SMTP port documented wrong

**`README.md:39`** says "configure your service to use localhost on port 8025"
but 8025 is the HTTP UI. SMTP is on port **1025**.

### 2.3 Health check only verifies HTTP

**`Dockerfile:22-23`** — `curl http://localhost:8025/` only checks the web UI.
If SMTP startup fails, the container is marked healthy.

**Fix:** Check both ports: `curl -sf http://localhost:8025/ && nc -z localhost 1025`

### 2.4 HTML body XSS — regex sanitization incomplete

**`http/httpd.go`** — Regex strips `<script>` and `on*=` attributes but can't
catch all vectors (data URIs, CSS imports, attribute injection).

**Fix:** Use a proper HTML sanitizer library (e.g., `bluemonday` for Go) or
DOMPurify on the client.

### 2.5 Attachment filename not quoted

**`http/httpd.go:441`** — `Content-Disposition: attachment; filename=my file.pdf`
is invalid. Filenames with spaces/special chars break the header.

**Fix:** `fmt.Sprintf("attachment; filename=%q", filename)`

---

## 3. Architecture Improvements

### 3.1 Storage layer (high impact)

| Issue | Details |
|-------|---------|
| Memory storage unimplemented | `storage_memory.go` — all methods return "unimplemented" |
| SQLite storage unimplemented | `storage_sqlite.go` — all methods return "unimplemented" |
| Config misleading | `default.json` lists 3 backends; only Filesystem works |
| No indexing | `filepath.Glob()` lists ALL files per query — O(n) |
| No caching | Every read re-parses the full .eml file |
| No file locking | Concurrent SMTP writes can corrupt data |

**Performance cliff:** 10K emails → 0.5-2s per search. 100K → unusable.

**Priority:** Implement SQLite backend with indexes on date, from, subject.

### 3.2 SMTP protocol gaps (high impact for testing tool)

| Missing feature | Why it matters |
|-----------------|---------------|
| STARTTLS | Can't test TLS-required senders (Gmail, Office365) |
| SMTP AUTH (inbound) | Can't test authenticated SMTP submission |
| Message SIZE limits | Can't test size-limit handling |
| Bounce/DSN simulation | Can't test bounce processing logic |
| Configurable failures | Can't test retry/error handling |
| VRFY/EXPN | Missing diagnostic SMTP commands |

**Comparison:** MailPit and smtp4dev both support TLS + AUTH.

### 3.3 Real-time updates (high impact for UX)

Current: JS polls `/api/emails/` every 5 seconds.

**Problems:**
- 5s delay between email send and UI update
- Polling generates traffic even when idle
- No notification if browser tab is in background

**Fix:** Add WebSocket or Server-Sent Events endpoint. Notify on new email,
delete, relay events.

---

## 4. Feature Roadmap

### Phase 1 — Critical fixes

| # | Task | Effort |
|---|------|--------|
| 1 | Fix graceful shutdown with signal handling | Low |
| 2 | Remove panic-restart loop | Low |
| 3 | Fix README SMTP port (1025 not 8025) | Low |
| 4 | Fix health check to verify SMTP port | Low |
| 5 | Quote attachment filenames in Content-Disposition | Low |
| 6 | Use `bluemonday` for proper HTML sanitization | Medium |
| 7 | Gate pprof behind `--debug` flag | Low |

### Phase 2 — Storage and performance

| # | Task | Effort |
|---|------|--------|
| 8 | Implement SQLite storage with indexes | High |
| 9 | Add email count caching (avoid full scan) | Medium |
| 10 | Implement in-memory storage for testing | Medium |
| 11 | Remove or clearly mark unimplemented backends in config | Low |
| 12 | Add file locking for concurrent SMTP writes | Medium |

### Phase 3 — SMTP protocol

| # | Task | Effort |
|---|------|--------|
| 13 | Add STARTTLS support with self-signed cert generation | High |
| 14 | Add inbound SMTP AUTH (accept any credentials) | Medium |
| 15 | Add configurable message SIZE limit | Low |
| 16 | Add bounce/DSN simulation mode | Medium |
| 17 | Add configurable failure injection (chaos testing) | Medium |

### Phase 4 — Real-time and API

| # | Task | Effort |
|---|------|--------|
| 18 | WebSocket/SSE for real-time email notifications | High |
| 19 | `GET /api/health` endpoint (HTTP + SMTP status) | Low |
| 20 | `GET /api/stats` endpoint (counts, uptime, storage) | Low |
| 21 | Wait-for-email API (`GET /api/emails/wait?query=...&timeout=30s`) | Medium |
| 22 | OpenAPI/Swagger specification | Medium |
| 23 | Environment variable configuration support | Medium |

### Phase 5 — Frontend UX

| # | Task | Effort |
|---|------|--------|
| 24 | Compose/send test email from UI | Medium |
| 25 | Keyboard shortcuts (j/k navigate, d delete, r release) | Low |
| 26 | Read/unread email tracking | Low |
| 27 | Email search in body content (currently broken) | Medium |
| 28 | Header-based search (X-Custom, Reply-To, etc.) | Low |
| 29 | Search results export (JSON/CSV) | Low |
| 30 | Email threading / conversation grouping | High |

### Phase 6 — Integration and deployment

| # | Task | Effort |
|---|------|--------|
| 31 | Docker Compose with TLS certs for STARTTLS testing | Low |
| 32 | Persistent volume mount for email data | Low |
| 33 | Webhook notifications on new email | Medium |
| 34 | API documentation in README | Medium |
| 35 | Configuration documentation | Low |

---

## 5. Competitive Comparison

| Feature | MailHog | MailPit | mock-my-mta |
|---------|---------|---------|-------------|
| SMTP TLS | Yes | Yes | No |
| SMTP AUTH | No | Yes | No (relay only) |
| Storage backends | MongoDB | SQLite | Filesystem only |
| WebSocket updates | Yes | Yes | No (polling) |
| Dark mode | No | Yes | Yes |
| MIME tree viewer | Yes | Yes | Yes |
| Search operators | Basic | Advanced | Good (Gmail-like) |
| Keyboard shortcuts | Yes | Yes | No |
| Bulk operations | No | Yes | Yes |
| .eml download | No | Yes | Yes |
| Raw headers view | No | Yes | Yes |
| E2E test suite | No | Some | Yes (45 tests) |
| HTML sanitization | No | Yes | Partial (regex) |
| Chaos/failure testing | No | No | No |
| Wait-for-email API | No | No | No |
| Binary size | ~14MB | ~30MB | ~6MB |

**mock-my-mta strengths:** Lightweight, strong test coverage, good search syntax,
MIME tree with actions, dark mode, bulk ops.

**mock-my-mta gaps:** No TLS/AUTH, no real-time, filesystem-only storage,
no compose, no chaos testing, incomplete security.

---

## 6. Known Limitations

- `message/rfc822` parts (forwarded emails) treated as binary blobs
- `multipart/digest` not specially handled
- Plain text body search disabled/broken
- No concurrent access safety (filesystem, no locking)
- Date-dependent e2e test (`newer_than:30d`) will break ~30 days after 2026-04-01
- Polling interval hardcoded to 5 seconds
- Page size hardcoded to 20 (not configurable)
- Date locale hardcoded to `'fr'` in `script.js`
