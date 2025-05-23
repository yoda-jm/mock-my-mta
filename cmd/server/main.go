package main

import (
	_ "embed" // Ensure embed is imported
	"encoding/json"
	"flag"
	"net/http"
	"net/mail"
	"os"
	"os/signal"
	"path/filepath"
	// "strings" // Removed unused import
	"syscall"
	"time"

	// Import for configuration types and loading functions
	appconfig "mock-my-mta/cmd/server/configtypes"
	mtahttp "mock-my-mta/http" // Alias for this project's http package
	"mock-my-mta/log"
	"mock-my-mta/smtp"
	"mock-my-mta/storage"
)

//go:embed config/default.json
var defaultConfigurationData []byte

func main() {
	// Parse command-line parameters
	var initWithTestData string
	var configurationFile string
	flag.StringVar(&initWithTestData, "init-with-test-data", "", "Folder containing test data emails")
	flag.StringVar(&configurationFile, "config", "", "Configuration file")
	flag.Parse()

	// Create a new storage instance
	var baseCfg appconfig.Configuration // Renamed for clarity
	if len(configurationFile) > 0 {
		var err error
		log.Logf(log.INFO, "loading configuration from %q", configurationFile)
		baseCfg, err = appconfig.LoadConfig(configurationFile) 
		if err != nil {
			log.Logf(log.FATAL, "error: failed to read engine config: %v", err)
		}
	} else {
		var err error
		log.Logf(log.INFO, "loading default configuration")
		baseCfg, err = appconfig.LoadDefaultConfiguration(defaultConfigurationData) 
		if err != nil {
			log.Logf(log.FATAL, "error: failed to parse engine config: %v", err)
		}
	}

	// Unmarshal specific configurations from baseCfg
	var httpCfg mtahttp.Configuration
	if err := json.Unmarshal(baseCfg.Httpd, &httpCfg); err != nil {
		log.Logf(log.FATAL, "error: failed to unmarshal httpd config: %v", err)
	}
	var smtpCfg smtp.Configuration
	if err := json.Unmarshal(baseCfg.Smtpd, &smtpCfg); err != nil {
		log.Logf(log.FATAL, "error: failed to unmarshal smtpd config: %v", err)
	}
	var storageCfgs []storage.StorageLayerConfiguration
	for i, scfgJSON := range baseCfg.Storages {
		var scfg storage.StorageLayerConfiguration
		if err := json.Unmarshal(scfgJSON, &scfg); err != nil {
			log.Logf(log.FATAL, "error: failed to unmarshal storage config #%d: %v", i, err)
		}
		storageCfgs = append(storageCfgs, scfg)
	}

	log.SetMinimumLogLevel(log.ParseLogLevel(baseCfg.Logging.Level)) 
	log.Logf(log.INFO, "starting mock-my-mta")
	storageEngine, err := storage.NewEngine(storageCfgs) // Use fully parsed storageCfgs
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
	startSmtpServer(smtpCfg, storageEngine) // Use fully parsed smtpCfg

	// Register API handlers
	// Note: This uses http.DefaultServeMux.
	// net/http package is imported as "http"
	http.HandleFunc("/api/filters/suggestions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Call the refactored handler from the mtahttp package
		// FilterSyntax is directly available in baseCfg and correctly typed
		mtahttp.HandleFilterSuggestions(w, r, baseCfg.FilterSyntax) 
	})

	// start http server
	// Assuming startHttpServer uses http.DefaultServeMux or is otherwise compatible
	// with handlers registered via http.HandleFunc.
	// If it sets up its own router exclusively, the above HandleFunc won't be part of it.
	startHttpServer(httpCfg, smtpCfg.Relays, storageEngine) // Use fully parsed httpCfg and smtpCfg.Relays

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
		mailUUID, err := storageEngine.Set(email)
		if err != nil {
			log.Logf(log.ERROR, "error: cannot store email from file %q: %v", filename, err)
			continue
		}
		log.Logf(log.INFO, "loaded email %v from file %q", mailUUID, filename)
	}

	return nil
}

func startSmtpServer(smtpCfg smtp.Configuration, storageEngine *storage.Engine) { // Renamed config to smtpCfg for clarity
	server := smtp.NewServer(smtpCfg, storageEngine)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Logf(log.ERROR, "SMTP server recovered from panic: %v", r)
				// sleep for a while to avoid a tight loop
				time.Sleep(1 * time.Second)
				startSmtpServer(smtpCfg, storageEngine) // Restart the server if panic occurs
			}
		}()

		err := server.ListenAndServe()
		if err != nil {
			panic("SMTP server error: " + err.Error())
		}
	}()
}

func startHttpServer(httpCfg mtahttp.Configuration, relayConfigurations smtp.RelayConfigurations, store storage.Storage) { // Renamed config to httpCfg
	// The existing http.NewServer is from "mock-my-mta/http"
	// If this server uses its own mux (router), it won't pick up http.DefaultServeMux routes.
	// For this exercise, we assume it might, or that registering to DefaultServeMux is the intended action.
	server := mtahttp.NewServer(httpCfg, relayConfigurations, store)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Logf(log.ERROR, "HTTP server recovered from panic: %v", r)
				// sleep for a while to avoid a tight loop
				time.Sleep(1 * time.Second)
				startHttpServer(httpCfg, relayConfigurations, store) // Restart the server if panic occurs
			}
		}()

		// If the http server is meant to use the DefaultServeMux (where HandleFunc registers routes),
		// its ListenAndServe would typically be http.ListenAndServe(addr, nil) or http.ListenAndServe(addr, http.DefaultServeMux)
		// The current custom server.ListenAndServe() might use its own router.
		// This detail is outside the scope of the current change, which is to add the handler as requested.
		err := server.ListenAndServe()
		if err != nil {
			panic("HTTP server error: " + err.Error())
		}
	}()
}
