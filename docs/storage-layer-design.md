# Storage Layer Design

## Overview

The storage engine uses a **multi-layer, scope-routed cascade** pattern. Each
layer declares which operation scopes it handles, and the engine routes each
API call to the appropriate subset of layers.

This enables:
- **Specialization** — each layer does what it's best at
- **Flexible deployment** — users configure exactly the layers they need
- **Write-through** — writes propagate to all layers that accept them
- **Read cascade** — reads try layers in order, first success wins
- **Graceful degradation** — unimplemented methods fall through automatically

## Architecture

```
                    ┌─────────────────────────────────┐
                    │           Engine                 │
                    │                                  │
                    │  readLayers:   [MEM, FS]         │
                    │  searchLayers: [SQLITE, FS]      │
                    │  writeLayers:  [MEM, SQLITE, FS] │
                    │  rawLayers:    [FS]               │
                    └────────┬────────────────┬────────┘
                             │                │
          ┌──────────────────┼────────────────┼──────────────────┐
          │                  │                │                  │
    ┌─────▼─────┐     ┌─────▼─────┐    ┌─────▼──────┐          │
    │  MEMORY   │     │  SQLITE   │    │ FILESYSTEM │          │
    │           │     │           │    │            │          │
    │ scope:    │     │ scope:    │    │ scope:     │          │
    │  read     │     │  search   │    │  all       │          │
    │  cache    │     │           │    │            │          │
    │           │     │           │    │            │          │
    │ Stores:   │     │ Stores:   │    │ Stores:    │          │
    │  parsed   │     │  indexed  │    │  raw .eml  │          │
    │  headers  │     │  metadata │    │  files     │          │
    │  bodies   │     │  (from,   │    │            │          │
    │  attach.  │     │   date,   │    │            │          │
    │  metadata │     │   subj.)  │    │            │          │
    └───────────┘     └───────────┘    └────────────┘
       volatile         persistent       persistent
       (rebuilt on       (survives        (source of
        restart)          restart)         truth)
```

## Operation Scopes

Each storage layer declares one or more scopes in its configuration.
The engine uses these to build per-scope routing tables at startup.

| Scope | Methods routed | Description |
|-------|---------------|-------------|
| `read` | GetEmailByID, GetBodyVersion, GetAttachments, GetAttachment | Single-email lookups by ID |
| `search` | SearchEmails, GetMailboxes | Filtered queries and aggregations |
| `write` | DeleteEmailByID, DeleteAllEmails | Mutations (plus Set which always goes to all writable layers) |
| `raw` | GetRawEmail | Raw email bytes for download/relay |
| `cache` | (receives writes via Set) | Volatile layer that receives write-through but rebuilds on restart |
| `all` | All of the above | Catch-all — layer handles everything |

### Scope resolution rules

1. `all` expands to every scope (`read`, `search`, `write`, `raw`, `cache`)
2. A layer with `cache` is included in write-through (receives `Set` calls)
   but is expected to be volatile — it rebuilds from the root layer via `load()`
3. The **last layer** in the config is the **root layer** (source of truth)
4. `Set()` always writes to all layers that have `write`, `cache`, or `all` scope

## Configuration

```json
{
  "storages": [
    {
      "type": "MEMORY",
      "scope": ["read", "cache"],
      "parameters": {}
    },
    {
      "type": "SQLITE",
      "scope": ["search"],
      "parameters": {
        "database": "emails.db"
      }
    },
    {
      "type": "FILESYSTEM",
      "scope": ["all"],
      "parameters": {
        "folder": "new-data",
        "type": "eml"
      }
    }
  ]
}
```

### Deployment profiles

**Minimal (filesystem only):**
```json
"storages": [
  { "type": "FILESYSTEM", "scope": ["all"], "parameters": { "folder": "data", "type": "eml" } }
]
```

**Fast reads (memory cache + filesystem):**
```json
"storages": [
  { "type": "MEMORY",     "scope": ["read", "cache"] },
  { "type": "FILESYSTEM", "scope": ["all"], "parameters": { "folder": "data", "type": "eml" } }
]
```

**Indexed search (sqlite + filesystem):**
```json
"storages": [
  { "type": "SQLITE",     "scope": ["search"], "parameters": { "database": "emails.db" } },
  { "type": "FILESYSTEM", "scope": ["all"], "parameters": { "folder": "data", "type": "eml" } }
]
```

**Full stack (recommended for large volumes):**
```json
"storages": [
  { "type": "MEMORY",     "scope": ["read", "cache"] },
  { "type": "SQLITE",     "scope": ["search"], "parameters": { "database": "emails.db" } },
  { "type": "FILESYSTEM", "scope": ["all"], "parameters": { "folder": "data", "type": "eml" } }
]
```

## Data Flow

### Write (new email via SMTP)

```
Engine.Set(message)
  │
  ├─ generate emailID (date-prefixed UUID)
  ├─ inject Date header if missing
  ├─ serialize message to []byte ONCE (parse-once optimization)
  │
  ├─ MEMORY.setWithID(id, rawBytes)     ← parse bytes → cache headers, bodies, attachments
  ├─ SQLITE.setWithID(id, rawBytes)     ← parse bytes → INSERT indexed metadata + store BLOB
  └─ FILESYSTEM.setWithID(id, rawBytes) ← write bytes directly to .eml file (zero parsing)
```

**Parse-once optimization:** The `mail.Message.Body` is an `io.Reader` that can only
be consumed once. The Engine serializes it to `[]byte` in `Set()`, then passes the
same immutable byte slice to all layers. Each layer creates its own
`bytes.NewReader(rawBytes)` only if it needs to parse. The Filesystem layer writes
raw bytes directly — no parsing needed for writes.

All layers with `write`, `cache`, or `all` scope receive the write.
A layer returning `unimplementedMethodInLayerError` is silently skipped.
A real error stops propagation and returns the error.

### Read by ID

```
Engine.GetEmailByID("2024-01-01T00:00:00Z-abc123")
  │
  ├─ MEMORY.GetEmailByID()  ← cache hit? return instantly
  │   └─ hit → return parsed EmailHeader
  │
  └─ FILESYSTEM.GetEmailByID()  ← cache miss, parse from disk
      └─ parse .eml → return EmailHeader
```

Layers tried in config order. First successful result returned.

### Search

```
Engine.SearchEmails("from:alice@test.com has:attachment", page=1, pageSize=20)
  │
  ├─ SQLITE.SearchEmails()  ← indexed query on from + has_attachments columns
  │   └─ hit → return results with pagination
  │
  └─ FILESYSTEM.SearchEmails()  ← fallback: scan all files, parse each, filter
      └─ slow O(n) scan → return results
```

### Delete

```
Engine.DeleteEmailByID("2024-01-01T00:00:00Z-abc123")
  │
  ├─ MEMORY.DeleteEmailByID()    ← remove from cache
  ├─ SQLITE.DeleteEmailByID()    ← DELETE FROM emails WHERE id = ?
  └─ FILESYSTEM.DeleteEmailByID() ← remove .eml file
```

All layers with `write` or `all` scope receive the delete.

### Initialization (startup)

```
Engine.load()
  │
  ├─ FILESYSTEM.load(nil)           ← root layer: create folder if needed
  │
  ├─ SQLITE.load(FILESYSTEM)        ← scan all filesystem emails, build index
  │   └─ for each email in root:
  │       INSERT INTO emails (id, from, to, subject, date, has_attachments, preview)
  │
  └─ MEMORY.load(FILESYSTEM)        ← parse all emails, populate cache
      └─ for each email in root:
          parse multipart → cache headers, bodies, attachments
```

The root layer (last in config) loads with `nil` — it initializes independently.
All other layers receive the root as `rootStorage` and can read from it to
populate themselves.

## Layer Specifications

### Memory Layer

**Purpose:** Parsed email cache for instant reads.

**Stores (in Go maps):**
- `map[string]EmailHeader` — parsed email headers indexed by ID
- `map[string]map[EmailVersionType]string` — decoded body versions per email
- `map[string][]AttachmentHeader` — attachment metadata per email
- `map[string]Attachment` — full attachment data (keyed by emailID/attachmentID)

**Characteristics:**
- Volatile — lost on restart, rebuilt via `load(rootStorage)`
- O(1) reads by ID
- No search capability (returns `unimplementedMethodInLayerError` for SearchEmails)
- Receives writes via `cache` scope to stay in sync during runtime

**Memory usage:** ~1KB per email header + body sizes. 10K emails ≈ 10-50MB RAM.

### SQLite Layer

**Purpose:** Indexed metadata for fast filtered searches.

**Schema:**
```sql
CREATE TABLE emails (
    id TEXT PRIMARY KEY,
    sender_name TEXT,
    sender_address TEXT,
    subject TEXT,
    date DATETIME,
    has_attachments BOOLEAN,
    preview TEXT,
    recipients TEXT  -- JSON array
);

CREATE INDEX idx_emails_date ON emails(date);
CREATE INDEX idx_emails_sender ON emails(sender_address);
CREATE INDEX idx_emails_subject ON emails(subject);
```

**Characteristics:**
- Persistent — survives restart (no need to rebuild from root)
- O(log n) indexed searches
- No body storage (returns `unimplementedMethodInLayerError` for GetBodyVersion)
- Handles SearchEmails, GetMailboxes, delete operations

### Filesystem Layer

**Purpose:** Raw email archive, source of truth.

**Stores:** One `.eml` file per email in a flat directory.

**Characteristics:**
- Persistent — the canonical data store
- O(n) for search (scans all files)
- Full capability — implements every method
- Used as `rootStorage` for other layers to hydrate from
- Parses emails on every read (no caching)

## Error Handling

### `unimplementedMethodInLayerError`

When a layer doesn't implement a method (either because it's outside its scope
or it's a stub), it returns this sentinel error type. The engine detects it and
continues to the next layer.

This serves as both:
1. **Routing mechanism** — layers only handle their declared scopes
2. **Safety net** — even if scope routing is misconfigured, the cascade still works

### Real errors

Any error that is NOT `unimplementedMethodInLayerError` stops the cascade and
is returned to the caller. This includes:
- File I/O errors
- Database errors
- Email parsing errors
- "Not found" errors (email ID doesn't exist in this layer)

### No layer implements the method

If ALL layers return `unimplementedMethodInLayerError`, the engine returns:
`"no storage layer implements <MethodName>"`

This is a configuration error — at least one layer (typically the `all`-scoped
root) should implement every method.

## Concurrency

The current implementation has no synchronization. For production use:
- Memory layer should use `sync.RWMutex` for map access
- SQLite layer handles concurrency via database locks
- Filesystem layer should use file-level locking for writes

For the primary use case (local development testing), single-writer concurrency
(one SMTP connection at a time) is acceptable.
