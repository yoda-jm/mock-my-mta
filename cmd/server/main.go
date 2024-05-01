package main

import (
	"flag"
	"net/mail"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"mock-my-mta/log"
	"mock-my-mta/storage"
)

const defaultConfiguration = `
{
	"smtpd": {
		"addr": ":8025",
		"relay-addr": ""
	},
	"httpd": {
		"addr": ":8080"
	},
	"storages": [
		{
			"type": "SQLITE",
			"parameters": {
				"database": "mock-my-mta.db"
			}
		},
		{
			"type": "MEMORY"
		},
		{
			"type": "MMM",
			"parameters": {
				"folder": "new-data"
			}
		}
	],
	"logging": {
		"level": "DEBUG"
	}
}`

func main() {
	// Parse command-line parameters
	var initWithTestData string
	flag.StringVar(&initWithTestData, "init-with-test-data", "", "Folder containing test data emails")
	flag.Parse()

	// Create a new storage instance
	config, err := parseConfiguration([]byte(defaultConfiguration))
	if err != nil {
		log.Logf(log.FATAL, "error: failed to parse engine config: %v", err)
	}
	log.SetMinimumLogLevel(log.ParseLogLevel(config.Logging.Level))
	log.Logf(log.INFO, "starting mock-my-mta")
	storageEngine, err := storage.NewEngine(config.Storages)
	if err != nil {
		log.Logf(log.FATAL, "error: failed to create storage: %v", err)
	}

	if len(initWithTestData) > 0 {
		log.Logf(log.INFO, "loading test data from %q", initWithTestData)
		err := loadTestData(storageEngine, initWithTestData)
		if err != nil {
			log.Logf(log.FATAL, "error: cannot load test data directory %q: %v:", initWithTestData, err)
		}
		// browse all the test data
		emailsHeaders, _, err := storageEngine.SearchEmails("", 1, -1)
		if err != nil {
			log.Logf(log.FATAL, "error: cannot get emails: %v", err)
		}
		for _, emailHeader := range emailsHeaders {
			log.Logf(log.INFO, "email %v: %v", emailHeader.ID, emailHeader)
			if emailHeader.HasAttachments {
				attachments, err := storageEngine.GetAttachments(emailHeader.ID)
				if err != nil {
					log.Logf(log.FATAL, "error: cannot get attachments for email %v: %v", emailHeader.ID, err)
				}
				for _, attachment := range attachments {
					log.Logf(log.INFO, "  attachment %v: %v", attachment.ID, attachment)
				}
			}
		}
	}

	// start smtp server
	startSmtpServer(config.Smtpd.Addr, storageEngine, config.Smtpd.RelayAddr)
	// start http server
	startHttpServer(config.Httpd.Addr, storageEngine)

	// Set up a signal handler to gracefully shutdown the servers on QUIT/TERM signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGQUIT, syscall.SIGTERM)

	<-quit // Wait for the QUIT/TERM signal
	log.Logf(log.INFO, "received QUIT/TERM signal. Shutting down servers...")

	// FIXME: shutdown servers
}

func loadTestData(storageEngine *storage.Engine, testDataDir string) error {
	// recursively find all eml files in the directory
	var filenames []string
	err := filepath.Walk(testDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".eml" {
			filenames = append(filenames, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, filename := range filenames {
		file, err := os.Open(filename)
		if err != nil {
			log.Logf(log.ERROR, "error: cannot read email from file %q: %v", filename, err)
			continue
		}
		email, err := mail.ReadMessage(file)
		if err != nil {
			log.Logf(log.ERROR, "error: cannot parse email from file %q: %v", filename, err)
			continue
		}
		err = storageEngine.Set(email)
		if err != nil {
			log.Logf(log.ERROR, "error: cannot store email from file %q: %v", filename, err)
			continue
		}
	}

	return nil
}

func startSmtpServer(addr string, storageEngine *storage.Engine, relayAddress string) {
	server := newSmtpServer(addr, storageEngine, relayAddress)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Logf(log.ERROR, "SMTP server recovered from panic:", r)
				startSmtpServer(addr, storageEngine, relayAddress) // Restart the server if panic occurs
			}
		}()

		err := server.ListenAndServe()
		if err != nil {
			panic("SMTP server error: " + err.Error())
		}
	}()
}

func startHttpServer(addr string, store storage.Storage) {
	server := newHttpServer(addr, store)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Logf(log.ERROR, "HTTP server recovered from panic:", r)
				startHttpServer(addr, store) // Restart the server if panic occurs
			}
		}()

		err := server.ListenAndServe()
		if err != nil {
			panic("HTTP server error: " + err.Error())
		}
	}()
}
