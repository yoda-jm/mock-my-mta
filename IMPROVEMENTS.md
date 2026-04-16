# Improvement Plan

Comprehensive audit of mock-my-mta — bugs, edge cases, missing features, and code quality issues.

---

## 1. Bugs (broken right now)

### 1.1 HTML-only emails show raw tags in preview

**File:** `storage/multipart/multipart.go:279-281`

When an email has no `text/plain` part, `GetPreview()` falls back to raw HTML without
stripping tags. The email list shows `<html><body><p>...` instead of readable text.

**Fix:** Strip HTML tags before using HTML content as preview fallback.

### 1.2 Emails with no Content-Type header — GetBody("plain-text") silently fails

**File:** `storage/multipart/multipart.go:80`

During parsing, the code defaults to `text/plain` when Content-Type is absent, but this
default is **not stored** in the leaf node headers. Later, `isPlainText()` checks the
actual (empty) header and returns `false`. The body becomes unreachable via the API.

**Fix:** Store the default `text/plain` content type in the leaf node headers when the
original email has no Content-Type.

### 1.3 Charset decoding not implemented

**File:** `storage/multipart/leaf_node.go:40-65` (FIXME at line 64)

`GetDecodedBody()` handles Content-Transfer-Encoding (base64, quoted-printable) but never
applies charset conversion from the Content-Type header. Emails with `charset=ISO-8859-1`,
`charset=windows-1252`, etc. display as mojibake.

**Fix:** Use `golang.org/x/text/encoding` to detect charset from Content-Type and convert
to UTF-8 after transfer decoding.

### 1.4 Typo in API response JSON field

**File:** `http/httpd.go:451`

The pagination response field is spelled `Paginnation` instead of `Pagination`. This is a
breaking change for any API consumer relying on the field name.

**Fix:** Rename to `Pagination`. Coordinate with frontend `script.js` if it reads this field.

---

## 2. Email Parsing Edge Cases

### 2.1 Verified working

| Case                                              | Status |
|---------------------------------------------------|--------|
| multipart/mixed > related > alternative + attach. | OK     |
| multipart/related without alternative             | OK     |
| Content-Disposition: inline vs attachment          | OK     |
| GetBody() for non-existent version                | OK     |
| CID resolution in deeply nested structures        | OK     |
| GetAttachments() at all nesting levels             | OK     |
| walkNodes() parent content-type tracking           | OK     |
| message/rfc822 (forwarded email as binary leaf)    | OK     |

### 2.2 Missing test coverage / edge cases to add

| Edge case                              | Issue                                                      |
|----------------------------------------|------------------------------------------------------------|
| Pure HTML email (no text/plain)        | Preview shows raw tags (bug 1.1)                           |
| Email with no Content-Type header      | Body unreachable (bug 1.2)                                 |
| Non-UTF-8 charset (ISO-8859-1)        | Mojibake (bug 1.3)                                         |
| RFC 2231 encoded filename              | Not decoded — `attachment_node.go:13-23` does naive split  |
| Content-Disposition: inline with CID   | Should be resolvable via CID, not listed as attachment      |
| Attachment with very long filename     | No truncation or sanitization                               |
| Multipart with missing closing boundary| `multipart.NewReader` throws unexpected EOF, no fallback   |
| Email with empty body                  | Should return empty string gracefully                       |

---

## 3. Header Encoding

### 3.1 Working

| Feature                        | Status  | Details                                  |
|--------------------------------|---------|------------------------------------------|
| RFC 2047 Subject decoding      | OK      | `decodeHeader()` with `mime.WordDecoder` |
| RFC 2047 From/To/CC decoding   | OK      | Applied before API response              |
| Multiple charsets in headers   | OK      | UTF-8, ISO-8859-1, etc.                 |
| Search on encoded headers      | OK      | Decoded before search                    |

### 3.2 Not working

| Feature                            | Status      | Location                        |
|------------------------------------|-------------|---------------------------------|
| RFC 2231 filename encoding         | Not handled | `attachment_node.go:13-23`      |
| RFC 2231 Content-Type parameters   | Not handled | Naive parsing                   |
| Body charset conversion            | Not handled | `leaf_node.go:64` FIXME         |

---

## 4. Frontend — Missing Features

### 4.1 High value (for a mail testing tool)

- **Raw headers view** — No way to see Message-ID, Received chain, Authentication-Results,
  DKIM, SPF, X-* headers. Essential for email debugging.
- **MIME structure tree** — No visualization of multipart nesting. Critical for debugging
  rendering issues in email clients.
- **Download .eml** — No way to save the original email file. Easy to implement via a new
  API endpoint returning the raw message.
- **New email notification** — Manual refresh only. Add polling or WebSocket support.

### 4.2 Nice to have

- Compose/send test email from UI
- Read/unread indicator
- Resizable left/right panes
- Dark mode
- Keyboard shortcuts (j/k navigation, d to delete, r to release)
- Email sorting by column (date, sender, subject)

---

## 5. Frontend — Display Issues

### 5.1 XSS in HTML body rendering

**File:** `http/static/script.js:517`

HTML body content from the API is injected via `shadowRoot.innerHTML` without sanitization.
Shadow DOM provides isolation but does not prevent script execution within the shadow tree.

**Fix:** Use DOMPurify or a similar library to sanitize HTML before injection.

### 5.2 Long content overflow

- Long email addresses break table layout (no `text-overflow: ellipsis` on sender column)
- Very long subjects overflow the email header view (no `word-wrap: break-word`)
- Many attachments create an unreadably long line (no wrapping or "show more")

### 5.3 Body version tabs

Currently rendered as inline text with commas. Should be styled as proper tabs or buttons
with clear active state.

---

## 6. Code Quality

### 6.1 Go backend

| Issue                                        | File                          | Line(s)  |
|----------------------------------------------|-------------------------------|----------|
| `Paginnation` typo in API response           | `http/httpd.go`               | 451      |
| `phytisalLayer` typo                         | `storage/engine.go`           | 21,25,28 |
| `recipents` typo                             | `storage/storage_filesystem.go`| 182     |
| `log.Fatalf` crashes server in library code  | `multipart/multipart_matcher.go`| 82     |
| Unsafe Content-Disposition (no quoting)      | `http/httpd.go`               | 380      |
| `context.TODO()` in Shutdown (no timeout)    | `http/httpd.go`               | 188      |
| Infinite panic restart loop                  | `cmd/server/main.go`          | 135-149  |
| FIXME: graceful shutdown not implemented     | `cmd/server/main.go`          | 90       |
| pprof endpoints always enabled               | `http/httpd.go`               | 70-84    |
| SQLite storage: all methods unimplemented    | `storage/storage_sqlite.go`   | 69       |
| Memory storage: all methods unimplemented    | `storage/storage_memory.go`   | 66       |
| Double-write on JSON encode error            | `http/httpd.go`               | 507-510  |
| Extra parens in route: `("/mailboxes")`      | `http/httpd.go`               | 62       |

### 6.2 Frontend (script.js)

| Issue                                          | Line(s)     |
|------------------------------------------------|-------------|
| ~9 undeclared global variables                 | 216,408,420,421,463,464,469,543,581 |
| Missing error handlers on 3 AJAX calls         | 210,457,530 |
| Error handlers only log (no user feedback)     | 384,434,608,361 |
| `data-toggle` (Bootstrap 4) vs Bootstrap 5     | index.html:23 |
| Deprecated `<center>` tag                      | script.js:589 |
| `generateMaiboxListItem` typo                  | script.js:381 (should be Mailbox) |
| Hardcoded `'fr'` locale                        | script.js:687 |
| `.replace(' ', '-')` only replaces first space | script.js:559 |
| Dead code: `#suggestions-dropdown` CSS         | styles.css:188-227 |
| Tooltip memory leak (never destroyed)          | script.js:148 |
| Commented-out code blocks                      | script.js:8-12,32-33,108-112 |

---

## 7. Implementation Priority

### Phase 1 — Fix bugs and add failing tests (this commit)

1. Add test fixtures for edge cases (pure HTML, no Content-Type, ISO-8859-1, RFC 2231)
2. Write failing unit tests for bugs 1.1–1.3
3. Write failing e2e test for HTML preview in email list

### Phase 2 — Fix the bugs

1. Strip HTML tags in `GetPreview()`
2. Store default Content-Type when header is missing
3. Implement charset decoding with `golang.org/x/text`
4. Fix `Paginnation` typo (coordinate with frontend)

### Phase 3 — Add raw headers & MIME tree

1. New API endpoint: `GET /api/emails/{id}/headers` (all raw headers)
2. New API endpoint: `GET /api/emails/{id}/mime-tree` (structure visualization)
3. New API endpoint: `GET /api/emails/{id}/download` (raw .eml file)
4. Frontend: Raw headers tab, MIME tree panel, download button

### Phase 4 — Frontend cleanup

1. Fix global variable declarations in script.js
2. Add missing AJAX error handlers
3. Fix Bootstrap 4/5 syntax mismatch
4. Remove dead CSS and commented-out code
5. Add DOMPurify for HTML body sanitization

### Phase 5 — Code quality

1. Fix all Go typos (Paginnation, phytisalLayer, recipents)
2. Replace `log.Fatalf` with error return
3. Sanitize Content-Disposition filenames
4. Add graceful shutdown with context timeout
5. Remove infinite panic restart loop
