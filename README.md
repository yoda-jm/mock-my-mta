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
- increase test coverage (and have an automatic report generated)
- github worflows for code quality, test coverage, binary build, ...
- add more relevant email examples, with corner cases tests
- add more data-testid in the frontend (in order to be easy to use with tools such as playwright)
- find a nice logo (and maybe a name) for the project
- fix search syntax by duration for requesting days/months/years
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

The goal of the project is to keep the code clean and minimal but still provinding all the expected features for such a software.

## License

This project is licensed under the [MIT License](LICENSE).

## Authors

This is the list of Authors:
- Vincent Le Ligeour (https://github.com/yoda-jm)