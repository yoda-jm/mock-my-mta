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
5. Configure your service sending emails to use localhost on port 8025

## Launching tests

To generate and view a test coverage report:
1. Generate the coverage profile: `go test -coverprofile=coverage.out ./...`
2. Open the report in a browser: `go tool cover -html=coverage.out -o coverage.html` (This will create `coverage.html` in your current directory).

## Running End-to-End Tests

End-to-end (E2E) tests are implemented using Playwright to simulate real user interactions with the web UI.

**Prerequisites:**
- Node.js and npm must be installed.

**Setup & Execution:**

1.  **Install Dependencies:**
    If you haven't already, or to ensure all project dependencies (including Playwright) are installed, run:
    ```bash
    npm install
    ```
    *(This assumes `playwright` and `@playwright/test` are listed in `package.json` devDependencies. If not, you might need `npm init playwright@latest --yes -- --quiet --browser=all` for a first-time setup).*

2.  **Start the Application Server for E2E Tests:**
    The server needs to be running with the specific test data located in the `e2e/testdata/emails` directory.
    ```bash
    go run ./cmd/server/ --init-with-test-data e2e/testdata/emails
    ```
    Keep this server running in a separate terminal.

3.  **Run Playwright Tests:**
    Execute the following command to run the E2E tests:
    ```bash
    npx playwright test
    ```

4.  **View Playwright HTML Report (Optional):**
    After the tests have run, you can view a detailed HTML report:
    ```bash
    npx playwright show-report
    ```

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
- github worflows for code quality, ...
- add more relevant email examples, with corner cases tests
- add more data-testid in the frontend (in order to be easy to use with tools such as playwright)
- find a nice logo (and maybe a name) for the project
- query typeahead when possible
- make Content-Security-Policy work when displaying email
- automatically embed images with cid (attached to the email)

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
