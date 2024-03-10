package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"mock-my-mta/log"
	"mock-my-mta/storage"
)

func main() {
	// Parse command-line parameters
	var smtpAddress, httpAddress, storageDir, testDataDir, relayAddress string
	flag.StringVar(&smtpAddress, "smtp-addr", ":8025", "Address of the SMTP server")
	flag.StringVar(&httpAddress, "http-addr", ":8080", "Address of the HTTP server")
	flag.StringVar(&storageDir, "storage", "", "Path to the storage directory")
	flag.StringVar(&testDataDir, "test-data", "", "Folder containing test data emails")
	flag.StringVar(&relayAddress, "relay-addr", "", "Address of the SMTP relay server")
	flag.Parse()

	if storageDir == "" {
		log.Logf(log.FATAL, "error: storage directory not provided")
	}

	// Create a new storage instance
	engineConfig := storage.EngineConfig{
		Storages: []storage.EngineLayerConfig{
			{
				Type: "MEMORY",
			},
			{
				Type: "MMM",
				Parameters: map[string]string{
					"folder": storageDir,
				},
			},
		},
	}
	store, err := storage.NewEngine(engineConfig)
	if err != nil {
		log.Logf(log.FATAL, "error: failed to create storage: %v", err)
	}

	if len(testDataDir) > 0 {
		so := storage.SortOption{
			Field:     storage.SortDateField,
			Direction: storage.Descending,
		}
		ids, err := storage.GetAll(store, so)
		if err != nil {
			log.Logf(log.FATAL, "error: cannot load test data into a non-empty storage")
		}
		if len(ids) > 0 {
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
	startSmtpServer(smtpAddress, store, relayAddress)
	// start http server
	startHttpServer(httpAddress, store)

	// Set up a signal handler to gracefully shutdown the servers on QUIT/TERM signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGQUIT, syscall.SIGTERM)

	<-quit // Wait for the QUIT/TERM signal
	log.Logf(log.INFO, "received QUIT/TERM signal. Shutting down servers...")
	// FIXME: shutdown servers
}

func startSmtpServer(addr string, store storage.Storage, relayAddress string) {
	server := newSmtpServer(addr, store, relayAddress)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Logf(log.ERROR, "SMTP server recovered from panic:", r)
				startSmtpServer(addr, store, relayAddress) // Restart the server if panic occurs
			}
		}()

		err := server.Start()
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

		err := server.Start()
		if err != nil {
			panic("HTTP server error: " + err.Error())
		}
	}()
}
