# MockMyMTA

A lightweight, local SMTP mock server for capturing and inspecting emails during development and testing.

[![GitHub](https://img.shields.io/badge/GitHub-yoda--jm%2Fmock--my--mta-blue?logo=github)](https://github.com/yoda-jm/mock-my-mta)
[![E2E Tests](https://img.shields.io/badge/tests-58%20e2e%20%2B%2090%2B%20unit-brightgreen)](https://github.com/yoda-jm/mock-my-mta/actions)

## What is MockMyMTA?

MockMyMTA captures all emails sent to it via SMTP and provides a web UI and REST API to browse, search, and inspect them. No email ever leaves your machine.

**Perfect for:**
- Local development — see emails your app sends without a real SMTP server
- E2E testing — verify email content with Playwright, Cypress, or any test framework
- CI/CD pipelines — wait for specific emails with the built-in wait-for-email API
- QA testing — inspect HTML rendering, attachments, MIME structure

## Quick Start

```bash
docker run -d -p 8025:8025 -p 1025:1025 vincentleligeour/mock-my-mta
```

- **Web UI:** http://localhost:8025
- **SMTP:** localhost:1025

Configure your application to send emails to `localhost:1025`. Every email appears instantly in the web UI.

## Features

- **SMTP** — STARTTLS, AUTH (PLAIN/LOGIN), configurable SIZE limits
- **Web UI** — Dark mode, Gmail-like search, bulk operations, real-time WebSocket updates
- **Email inspection** — HTML/text/raw body tabs, raw headers, MIME structure tree, CID image preview
- **Attachments** — Download, inline preview, .eml file download
- **Search** — `from:`, `subject:`, `has:attachment`, `before:`, `after:`, `older_than:`, `newer_than:`, free text
- **API** — Full REST API, WebSocket events, wait-for-email endpoint for CI/CD
- **Chaos testing** — Configurable reject rate, delay, bounce simulation via settings modal
- **Storage** — Multi-layer (memory cache + filesystem), scoped routing
- **Deep links** — Shareable URLs for searches, emails, and tabs

## Wait-for-Email API (CI/CD)

```bash
# Block until a matching email arrives (no sleep needed):
curl "http://localhost:8025/api/emails/wait?query=subject:Welcome&timeout=30s"

# Response includes a URL to view the email:
# {"email":{...}, "total_matches":1, "url":"http://localhost:8025/#/email/..."}
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MOCKMYMTA_SMTP_ADDR` | `:1025` | SMTP listen address |
| `MOCKMYMTA_HTTP_ADDR` | `:8025` | HTTP listen address |
| `MOCKMYMTA_LOG_LEVEL` | `INFO` | Log level (DEBUG, INFO, WARNING, ERROR) |
| `MOCKMYMTA_SMTP_MAX_MESSAGE_SIZE` | `0` (unlimited) | Max email size in bytes |

### Docker Compose

```yaml
services:
  mock-smtp:
    image: vincentleligeour/mock-my-mta
    ports:
      - "8025:8025"
      - "1025:1025"
    volumes:
      - email-data:/app/new-data
    environment:
      MOCKMYMTA_LOG_LEVEL: DEBUG

volumes:
  email-data:
```

### With config file

```bash
docker run -d -p 8025:8025 -p 1025:1025 \
  -v ./config.json:/app/config.json \
  vincentleligeour/mock-my-mta ./server --config config.json
```

## Ports

| Port | Protocol | Description |
|------|----------|-------------|
| 1025 | SMTP | Email reception (STARTTLS available) |
| 8025 | HTTP | Web UI and REST API |

## Architecture

- Single Go binary (~6MB)
- Multi-layer storage engine (memory cache + filesystem)
- Pure Go — no CGo dependencies
- Multi-arch image: `linux/amd64` and `linux/arm64`

## Links

- **Source:** https://github.com/yoda-jm/mock-my-mta
- **Issues:** https://github.com/yoda-jm/mock-my-mta/issues
- **License:** MIT
