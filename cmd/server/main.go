package main

import (
	"context"
	"flag"
	"net/mail"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	mtahttp "mock-my-mta/http"
	"mock-my-mta/log"
	"mock-my-mta/smtp"
	"mock-my-mta/storage"
)

func main() {
	var initWithTestData string
	var configurationFile string
	flag.StringVar(&initWithTestData, "init-with-test-data", "", "Folder containing test data emails")
	flag.StringVar(&configurationFile, "config", "", "Configuration file")
	flag.Parse()

	// Load configuration
	var config Configuration
	if len(configurationFile) > 0 {
		var err error
		log.Logf(log.INFO, "loading configuration from %q", configurationFile)
		config, err = LoadConfig(configurationFile)
		if err != nil {
			log.Logf(log.FATAL, "error: failed to read engine config: %v", err)
		}
	} else {
		var err error
		log.Logf(log.INFO, "loading default configuration")
		config, err = LoadDefaultConfiguration()
		if err != nil {
			log.Logf(log.FATAL, "error: failed to parse engine config: %v", err)
		}
	}

	// Environment variable overrides
	applyEnvOverrides(&config)

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
			log.Logf(log.FATAL, "error: cannot load test data directory %q: %v", initWithTestData, err)
		}
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

	// Start servers
	smtpServer := smtp.NewServer(config.Smtpd, storageEngine)
	httpServer := mtahttp.NewServer(config.Httpd, config.Smtpd.Relays, storageEngine)

	// Wire SMTP → WebSocket notification
	smtpServer.SetOnNewEmail(func(emailID string) {
		mtahttp.BroadcastEvent("new_email", map[string]string{"id": emailID})
	})
	// Wire SMTP behavior settings from HTTP settings API
	smtpServer.SetGetBehavior(func() smtp.SmtpBehavior {
		s := mtahttp.GetSmtpSettings()
		return smtp.SmtpBehavior{
			RejectRate:    s.RejectRate,
			RejectMessage: s.RejectMessage,
			DelayMs:       s.DelayMs,
			BounceRate:    s.BounceRate,
			BounceMessage: s.BounceMessage,
		}
	})

	go func() {
		if err := smtpServer.ListenAndServe(); err != nil {
			log.Logf(log.FATAL, "SMTP server error: %v", err)
		}
	}()

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Logf(log.FATAL, "HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown on QUIT/TERM/INT signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Logf(log.INFO, "shutting down servers (5s timeout)...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Logf(log.ERROR, "HTTP server shutdown error: %v", err)
	}
	log.Logf(log.INFO, "servers stopped")
}

func loadTestData(storageEngine *storage.Engine, testDataDir string) error {
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

// applyEnvOverrides lets environment variables override JSON config values.
// Env var names: MOCKMYMTA_SMTP_ADDR, MOCKMYMTA_HTTP_ADDR, MOCKMYMTA_LOG_LEVEL, etc.
func applyEnvOverrides(config *Configuration) {
	if v := os.Getenv("MOCKMYMTA_SMTP_ADDR"); v != "" {
		config.Smtpd.Addr = v
	}
	if v := os.Getenv("MOCKMYMTA_HTTP_ADDR"); v != "" {
		config.Httpd.Addr = v
	}
	if v := os.Getenv("MOCKMYMTA_LOG_LEVEL"); v != "" {
		config.Logging.Level = v
	}
	if v := os.Getenv("MOCKMYMTA_HTTP_DEBUG"); v == "true" || v == "1" {
		config.Httpd.Debug = true
	}
	if v := os.Getenv("MOCKMYMTA_SMTP_MAX_MESSAGE_SIZE"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			config.Smtpd.MaxMessageSize = size
		}
	}
}
