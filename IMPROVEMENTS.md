# Improvement Plan

Status of the mock-my-mta project after the audit and fix cycle.

---

## 1. Completed

### 1.1 Bugs fixed

| Bug | Fix | Commit |
|-----|-----|--------|
| HTML-only emails show raw tags in preview | `stripHTMLTags()` in `GetPreview()` | 5731d84 |
| No Content-Type header → empty body | Default `text/plain` injected during parsing | 5731d84 |
| ISO-8859-1 charset → mojibake | `golang.org/x/text` charset conversion in `GetDecodedBody()` | 5731d84 |
| RFC 2231 filename not decoded | `mime.ParseMediaType` in `GetFilename()` | 5731d84 |
| `Paginnation` typo in API JSON | Renamed to `Pagination` | 14052da |
| `phytisalLayer` / `recipents` / `GetRaEmail` typos | Fixed across Go codebase | 14052da |
| Bootstrap 4 `data-toggle` with Bootstrap 5 | Changed to `data-bs-toggle` | 14052da |
| ~9 undeclared global variables in script.js | Added `const`/`let` declarations | 14052da |
| Dead `#suggestions-dropdown` CSS (40 lines) | Removed | 14052da |
| Deprecated `<center>` HTML tag | Replaced with `text-center` CSS class | 14052da |
| 1x1px invisible test images | Replaced with 40x40px colored PNGs | 2181a18 |

### 1.2 Features added

| Feature | Details |
|---------|---------|
| Raw headers view | `GET /api/emails/{id}/headers` + "headers" tab in UI |
| .eml download | `GET /api/emails/{id}/download` + download button in toolbar |
| MIME structure tree | `GET /api/emails/{id}/mime-tree` + "mime-tree" tab with badges |
| MIME preview modal | CID content previewed in-page modal instead of new tab |
| MIME tree actions | "view" switches to body tab, "preview" opens modal, "open" for CID |
| Proper nav tabs | Bootstrap `nav-tabs` with icons for all body versions |
| Smart external images toggle | Only shown for HTML/watch-html body tabs |
| E2E screenshot capture | `takeAndAttachScreenshot()` + `screenshotLocator()` on all tests |
| GitHub Pages report | Playwright report deployed to GitHub Pages on push to main |
| Dedicated e2e CI workflow | `e2e.yml` with summary table and artifact uploads |

### 1.3 Code quality

| Item | Details |
|------|---------|
| Go 1.25 | Upgraded from 1.21 for `golang.org/x/text` compatibility |
| GitHub Actions Node.js 24 | All actions bumped to Node.js 24-compatible versions |
| Page object model for e2e | InboxPage + 8 section classes |
| Docker e2e setup | `docker-compose.yml` with healthcheck-gated e2e container |

---

## 2. Test Coverage Summary

### 2.1 Current numbers

- **42 e2e tests** across 3 spec files (all passing)
- **57+ Go unit tests** across 8 test files (all passing)
- **14 API endpoints** — 13/14 covered by e2e (93%), 6/14 by unit tests (43%)

### 2.2 Gaps — no test coverage

| Gap | Severity | What's missing |
|-----|----------|----------------|
| API error responses (404, 400, 500) | High | No unit tests for error paths in any HTTP handler |
| Email-not-found scenarios | High | No test for requesting non-existent email ID |
| Attachment-not-found scenarios | Medium | No test for invalid attachment ID |
| CID-not-found scenarios | Medium | `getPartByCID` only tested happy path |
| Pagination edge cases | Medium | No test for page=0, negative, very large |
| Invalid relay request body | Medium | No test for malformed JSON in POST relay |
| Unsupported/unknown charsets | Low | Only ISO-8859-1 tested, not windows-1252, Shift_JIS |
| XSS in HTML body rendering | Low | `shadowRoot.innerHTML` not sanitized |
| pprof endpoints | Low | Always exposed, no auth, no tests |
| SQLite/Memory storage | Low | Completely unimplemented, only config tested |

### 2.3 Weak coverage — happy path only

| Area | Current state | Improvement needed |
|------|---------------|-------------------|
| Release/relay flow | Modal UI tested, POST relay has unit tests | E2E for actual relay submission, error modal |
| Search filters | All operators tested individually | Error cases, malformed queries, special chars in values |
| SMTP ingestion | Basic success + storage error | Large messages, concurrent delivery, malformed emails |
| Body version switching | Tab clicks tested | Missing version returns empty, network error |
| Autocomplete | Suggestion display + Tab tested | API failure, very long input, special chars |

---

## 3. Remaining Work

### 3.1 Phase 4 — Test hardening (high priority)

1. **HTTP handler unit tests** — Test error responses for each endpoint:
   - `getEmailByID` with non-existent ID → 404
   - `getBodyVersion` with invalid version → 400
   - `getAttachmentContent` with invalid attachment → 404
   - `getPartByCID` with non-existent CID → 404
   - `relayMessage` with malformed JSON → 400
   - `parsePageParameters` with invalid values → 400

2. **E2E error scenario tests**:
   - Search with empty results shows "No emails found" message
   - Release modal with no relay configured shows appropriate feedback
   - Navigate to email that was deleted → handle gracefully

3. **Charset edge cases**:
   - windows-1252 body decoding
   - Unknown charset fallback behavior
   - Mixed charset in multipart email

### 3.2 Phase 5 — Security hardening

1. **HTML body sanitization** — Add DOMPurify or server-side sanitization
   before rendering HTML email bodies in `shadowRoot.innerHTML`
2. **Content-Disposition filename quoting** — `httpd.go:380` directly
   concatenates filename without RFC 5987 quoting
3. **pprof access control** — Debug endpoints always exposed; consider
   gating behind a flag or removing in production builds
4. **Graceful shutdown** — `context.TODO()` in `Shutdown()` should use
   `context.WithTimeout()`

### 3.3 Phase 6 — Architecture improvements

1. **Implement Memory storage layer** — Currently returns "unimplemented"
2. **Implement SQLite storage layer** — Currently returns "unimplemented"
3. **Graceful shutdown with signal handling** — `cmd/server/main.go:90`
   has a FIXME; servers start but never shut down cleanly
4. **Remove infinite panic restart loop** — `cmd/server/main.go:135-149`
   restarts on panic with 1s sleep; masks real issues

### 3.4 Phase 7 — Nice-to-have features

| Feature | Effort | Impact |
|---------|--------|--------|
| New email notification (polling/WebSocket) | Medium | High — manual refresh only today |
| Compose/send test email from UI | Medium | High — can only release existing emails |
| Keyboard shortcuts (j/k, d, r) | Low | Medium — power user productivity |
| Read/unread indicator | Low | Low — visual state tracking |
| Resizable left/right panes | Low | Low — fixed 250px sidebar |
| Dark mode | Medium | Low — light theme only |
| Email sorting by column headers | Low | Medium — date-sorted only |

---

## 4. Known Limitations

- **message/rfc822 parts** (forwarded emails) are treated as binary blobs,
  not recursively parsed. This is by design.
- **multipart/digest** not specially handled.
- **SMTP TLS** not supported (plain SMTP only on port 1025).
- **Concurrent access** — filesystem storage has no locking; fine for
  development/testing use but not production.
- **Date-dependent e2e test** — `newer_than:30d` test depends on
  `email_dated_recent.eml` being within 30 days of the test run date
  (2026-04-01). Will need updating if tests run much later.
