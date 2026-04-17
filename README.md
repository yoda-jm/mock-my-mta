# Project Name

## Description

This project provides a MTA (Mail Transfer Agent) mock. The goal is to provide
a SMTP server that can store and visualize email sent by another application.

It is very useful in the following cases:
- Your application does not have access to a SMTP server.
- Your development environments do not have access to a SMTP sever.
- You don't want your test environment to send any email to the outside world.
- You want to be able to fully test your application and the email sent using a non regression tool (playwright for example).
- You want to be able to capture all the email sent by your application.

The development has been inspired by MailHog (https://github.com/mailhog/MailHog) but with the following improvements:
- keep the code base small and in 1 repository
- keep the code as much as possible simple
- have a way to have both the fast memory browsing and the filesystem persistance without requiring a third party service like a database
- keep the basic features (MTA mocking, 1-page http GUI, releasing emails manually/automatically)

## Features (achieved and expected)

- A SMTP server (see https://github.com/chrj/smtpd)
- A REST api to access the service
- A HTTP GUI for displaying emails (using the REST api)
- A search language for finding emails (GMail-like syntax)
- A SMTP client to forward/release email to another MTA
- A multi-layer storage engine (offering both performances and persistance)
- A simple 1-binary service

## Installation

To install and run this project, follow these steps:

1. Clone the repository: `git clone https://github.com/yoda-jm/mock-my-mta.git && cd mock-my-mta`
2. Launch tests `go test ./... -v`
3. Launch with some testing emails: `go run ./cmd/server/ --init-with-test-data testdata`
4. Connect your browser to http://localhost:8080
5. Configure your service sending emails to use localhost on SMTP port 1025 (the web UI is on port 8025)

## Architecture

- [Storage Layer Design](docs/storage-layer-design.md) — multi-layer,
  scope-routed cascade architecture for memory caching, SQLite indexing,
  and filesystem persistence.

## Launching tests

To generate and view a test coverage report:
1. Generate the coverage profile: `go test -coverprofile=coverage.out ./...`
2. Open the report in a browser: `go tool cover -html=coverage.out -o coverage.html` (This will create `coverage.html` in your current directory).

## Running End-to-End Tests

End-to-end (E2E) tests are implemented using Playwright to simulate real user interactions with the web UI.

**Prerequisites:**
- Node.js and npm must be installed.

**Setup & Execution:**

### With Docker (recommended — zero host dependencies)

```bash
# Build + run the full suite inside Docker Compose, then exit:
npm run docker:test
# (equivalent to: docker compose down --remove-orphans && docker compose run --rm e2e)

# Serve the interactive HTML report from the last run:
npm run report:serve   # opens http://localhost:9323
```

The `docker:test` script starts the Go server in one container, waits for
the healthcheck to pass, then runs Playwright in a second container.

### Locally (requires Go ≥ 1.21 and Node.js ≥ 18)

```bash
# 1. Install Node dependencies and the Chromium browser
npm ci
npx playwright install chromium --with-deps

# 2. Build the server, start it with the E2E test dataset
go build -o server ./cmd/server/
./server --init-with-test-data e2e/testdata/emails &

# 3. Run the tests
npx playwright test

# 4. Open the interactive HTML report
npm run report:serve   # opens http://localhost:9323
```

### Test structure

| File | Purpose | Run order |
|------|---------|-----------|
| `e2e/email_display.spec.js` | Decoding (QP, base64), CID images, header fields, autocomplete | 1st |
| `e2e/email_features.spec.js` | UI features + all search filter operators | 2nd |
| `e2e/email_interaction.spec.js` | Full interaction + destructive (delete) tests | Last |

Files run in alphabetical order. Destructive tests are placed last within
their file; `email_interaction.spec.js` is last overall.

### Test data

Crafted `.eml` fixtures live in `e2e/testdata/emails/`. Key files:

| File | Covers |
|------|--------|
| `multipart_alternative_watch.eml` | Apple Watch HTML body version |
| `multipart_mixed_related_alternative_attachments.eml` | Multiple attachments |
| `cid_image_only.eml` | CID image rewriting to API endpoint |
| `email_with_external_image.eml` | External image hide/show toggle |
| `email_various_headers.eml` | Display name, CC, ID header fields |
| `email_qp_french.eml` | Quoted-printable UTF-8 decoding |
| `email_base64_body.eml` | Base64 body decoding |
| `email_unique_from.eml` | `from:` filter isolation |
| `email_dated_old.eml` | `before:` / `older_than:` filters (dated 2020) |
| `email_dated_recent.eml` | `after:` / `newer_than:` filters (dated 2026-04-01) |
| `email_with_specialchars.eml` | Special character + quoted phrase search |
| `email1.eml`, `email2.eml` | `mailbox:` filter (recipient1@example.com) |
| `email3–7.eml` | Pagination (total > 20 emails) |

### CI — GitHub Actions

A dedicated workflow (`.github/workflows/e2e.yml`) runs on every push and
pull request to `main`. It:

1. Builds the Go server binary natively.
2. Installs Playwright's Chromium browser.
3. Waits for the server to pass a health check before starting tests.
4. Emits inline **GitHub annotations** on failure — visible on the commit/PR page.
5. Posts a **pass/fail summary table** to the Actions run summary tab.
6. Uploads the **interactive HTML report** as a downloadable artifact
   (`playwright-report`, kept for 30 days).

## Additional information

If you want more emails to test performance and various emails, you can put `.eml` email files
in the folder `testdata` (it is scanned recursively when using  `--init-with-test-data testdata`).
Here is a list of resources providing such emails:
- https://github.com/mikel/mail/tree/master/spec/fixtures/emails
- ENRON extract (4.5MB): https://web.archive.org/web/20110307220813/http://bailando.sims.berkeley.edu/enron/enron_with_categories.tar.gz
- full ENRON (1.7GB): https://web.archive.org/web/20110307220813/http://www-2.cs.cmu.edu/~enron/enron_mail_030204.tar.gz

## TODOs

There is still a lot of things that should be done:
- improve GUI (general layout, button interactions, ...)
- implement new storage layers (memory, sqlite, other SQL, mongo, ...)
- github workflows for code quality, ...
- find a nice logo (and maybe a name) for the project
- make Content-Security-Policy work when displaying email

Done:
- ✅ add more relevant email examples, with corner cases tests
- ✅ add more data-testid in the frontend (in order to be easy to use with tools such as playwright)
- ✅ query typeahead / autocomplete for search
- ✅ automatically rewrite CID image references to the API endpoint

## Contributing

Contributions are welcome! If you'd like to contribute to this project, please follow these guidelines:

1. Fork the repository.
2. Create a new branch: `git checkout -b feature/your-feature-name`
3. Make your changes and commit them: `git commit -m 'Add some feature'`
4. Push to the branch: `git push origin feature/your-feature-name`
5. Submit a pull request.

**Frontend Development Note:**
To ensure robust and maintainable end-to-end tests, `data-testid` attributes have been systematically added to interactive HTML elements in the frontend (`http/static/index.html` and dynamically in `http/static/script.js`). When making changes to the UI, please:
- Utilize existing `data-testid` attributes for selecting elements in tests.
- Add new `data-testid` attributes to new interactive elements.
- Ensure `data-testid` attributes remain consistent and descriptive.

The goal of the project is to keep the code clean and minimal but still provinding all the expected features for such a software.

## License

This project is licensed under the [MIT License](LICENSE).

## Authors

This is the list of Authors:
- Vincent Le Ligeour (https://github.com/yoda-jm)
