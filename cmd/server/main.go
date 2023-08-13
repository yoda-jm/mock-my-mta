package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"mock-my-mta/email"
	"mock-my-mta/log"
	"mock-my-mta/storage"
)

func main() {
	// Parse command-line parameters
	var smtpAddress, httpAddress, storageDir, testDataDir string
	flag.StringVar(&smtpAddress, "smtp-addr", ":8025", "Address of the SMTP server")
	flag.StringVar(&httpAddress, "http-addr", ":8080", "Address of the HTTP server")
	flag.StringVar(&storageDir, "storage", "", "Path to the storage directory")
	flag.StringVar(&testDataDir, "test-data", "", "Folder containing test data emails")
	flag.Parse()

	if storageDir == "" {
		log.Logf(log.FATAL, "error: storage directory not provided")
	}

	// Create a new storage instance
	store, err := storage.NewStorage(storageDir)
	if err != nil {
		log.Logf(log.FATAL, "error: failed to create storage: %v", err)
	}

	if len(testDataDir) > 0 {
		so := storage.SortOption{
			Field:     storage.SortDateField,
			Direction: storage.Descending,
		}
		if len(store.GetAll(so)) > 0 {
			log.Logf(log.FATAL, "error: cannot load test data into a non-empty storage")
		}
		err = loadTestData(store, testDataDir)
		if err != nil {
			log.Logf(log.FATAL, "error: cannot load test data directory %q: %v:", testDataDir, err)
		}
	}

	const testStorage = false
	if testStorage {
		// Find emails with matching criteria
		testFind(store)
		// Print the content of the storage
		testPrintAll(store)
	}

	// start smtp server
	startSmtpServer(smtpAddress, store)
	// start http server
	startHttpServer(httpAddress, store)

	// Set up a signal handler to gracefully shutdown the servers on QUIT/TERM signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGQUIT, syscall.SIGTERM)

	<-quit // Wait for the QUIT/TERM signal
	log.Logf(log.INFO, "received QUIT/TERM signal. Shutting down servers...")
	// FIXME: shutdown servers
}

func testFind(store *storage.Storage) {
	for _, direction := range []storage.SortType{storage.Ascending, storage.Descending} {
		so := storage.SortOption{
			Field:     storage.SortDateField,
			Direction: direction,
		}
		searchPattern := "Email 1"
		log.Logf(log.DEBUG, "searching for emails with subject %q", searchPattern)
		mo := email.MatchOption{Field: email.MatchSubjectField, Type: email.ExactMatch, CaseSensitive: true}
		foundEmails := store.Find(mo, so, searchPattern)
		log.Logf(log.DEBUG, "found %d emails", len(foundEmails))
		for _, id := range foundEmails {
			log.Logf(log.DEBUG, "found email with ID: %v", id)
		}
	}
}

func testPrintAll(store *storage.Storage) {
	for _, direction := range []storage.SortType{storage.Ascending, storage.Descending} {
		so := storage.SortOption{
			Field:     storage.SortDateField,
			Direction: direction,
		}
		so.Direction = direction
		// Get email data by UUID
		log.Logf(log.DEBUG, "getting all emails from storage")
		uuids := store.GetAll(so)
		if len(uuids) == 0 {
			log.Logf(log.DEBUG, "no UUID found in storage")
		} else {
			log.Logf(log.DEBUG, "found %d emails", len(uuids))
			for _, uuid := range uuids {
				emailData, found := store.Get(uuid)
				if found {
					log.Logf(log.DEBUG, "retrieved email data for UUID:%v, received at %v", emailData.ID, emailData.ReceivedTime)
					log.Logf(log.DEBUG, "subject: %v", emailData.Email.GetSubject())
					versions := emailData.Email.GetVersions()
					log.Logf(log.DEBUG, "body verions count: %v", len(versions))
					for _, version := range versions {
						body, err := emailData.Email.GetBody(version)
						if err != nil {
							log.Logf(log.DEBUG, "cannot get body for version %q", version)
						} else {
							log.Logf(log.DEBUG, "- %q: %v bytes", version, len(body))
						}
					}
					attachmentIDs := emailData.Email.GetAttachments()
					if len(attachmentIDs) > 0 {
						log.Logf(log.DEBUG, "%v attachments", len(attachmentIDs))
						for _, attachmentID := range attachmentIDs {
							attachment, found := emailData.Email.GetAttachment(attachmentID)
							if found {
								log.Logf(log.DEBUG, "- attachment (id=%v, type=%v, filename=%q, length=%v)", attachment.GetID(), attachment.GetMediaType(), attachment.GetFilename(), len(attachment.GetContent()))
							} else {
								log.Logf(log.DEBUG, "email attachment not found for UUID:%v", attachmentID)
							}
						}
					}
				} else {
					log.Logf(log.DEBUG, "email data not found for UUID:%v", emailData.ID)
				}
			}
		}
	}
}

func startSmtpServer(addr string, store *storage.Storage) {
	server := newSmtpServer(addr, store)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Logf(log.ERROR, "SMTP server recovered from panic:", r)
				startSmtpServer(addr, store) // Restart the server if panic occurs
			}
		}()

		err := server.Start()
		if err != nil {
			panic("SMTP server error: " + err.Error())
		}
	}()
}

func startHttpServer(addr string, store *storage.Storage) {
	server := newHttpServer(addr, store)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Logf(log.ERROR, "HTTP server recovered from panic:", r)
				startHttpServer(addr, store) // Restart the server if panic occurs
			}
		}()

		err := server.Start()
		if err != nil {
			panic("HTTP server error: " + err.Error())
		}
	}()
}

func loadTestData(store *storage.Storage, testDataDir string) error {
	files, err := ioutil.ReadDir(testDataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".txt" {
			continue
		}

		filePath := filepath.Join(testDataDir, file.Name())
		log.Logf(log.INFO, "loading file %q", filePath)
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		err = store.Set(content)
		if err != nil {
			return err
		}
	}

	return nil
}
