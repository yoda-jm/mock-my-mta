package configuration

import (
	_ "embed"
	"encoding/json"
	// Note: These imports might cause cycles if http, smtp, storage depend on configuration
	// and configuration now tries to import them.
	"io/ioutil"
	"log"
	// "mock-my-mta/http"  // Removed to break import cycle
	// "mock-my-mta/smtp"  // Removed to break import cycle
	// "mock-my-mta/storage" // Removed to break import cycle
	"os"
	"path/filepath"
	"fmt" // Added for fmt.Errorf
)

// FilterSyntaxEntry defines the structure for each filter command syntax help.
type FilterSyntaxEntry struct {
	Command     string `json:"command"`
	Suggestion  string `json:"suggestion"`
	Description string `json:"description"`
}

// Configuration holds the application configuration.
// Fields that would cause import cycles are stored as json.RawMessage
// and should be unmarshalled by the main application.
type Configuration struct {
	Smtpd        json.RawMessage       `json:"smtpd"`
	Httpd        json.RawMessage       `json:"httpd"`
	Storages     []json.RawMessage     `json:"storages"` // Each element is a raw message
	Logging      LoggingConfiguration  `json:"logging"`
	FilterSyntax []FilterSyntaxEntry   `json:"filter_syntax,omitempty"`
}

type LoggingConfiguration struct {
	Level string `json:"level"`
}

func parseConfiguration(data []byte) (Configuration, error) {
	var config Configuration
	err := json.Unmarshal(data, &config)
	if err != nil {
		return Configuration{}, err
	}
	return config, nil
}

// LoadConfig reads the main configuration file and then attempts to load
// the filter_syntax.json file from the same directory.
// If filter_syntax.json is not found or is invalid, it logs an error and proceeds.
func LoadConfig(configPath string) (Configuration, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return Configuration{}, err
	}
	cfg, err := parseConfiguration(data)
	if err != nil {
		// If the main config itself fails to parse, we should return that error.
		return cfg, err
	}

	// Construct path to filter_syntax.json, expected to be in the same directory
	// as the main configuration file.
	filterSyntaxJSONPath := filepath.Join(filepath.Dir(configPath), "filter_syntax.json")

	filterSyntaxData, err := ioutil.ReadFile(filterSyntaxJSONPath)
	if err != nil {
		// Log and continue if filter_syntax.json is not found.
		// The server can start without this, filter help will just be unavailable.
		log.Printf("INFO: Optional file filter_syntax.json not found at %s, or error reading it: %v. Filter syntax help will be unavailable.", filterSyntaxJSONPath, err)
		cfg.FilterSyntax = []FilterSyntaxEntry{} // Ensure it's empty
		return cfg, nil
	}

	var filterSyntaxEntries []FilterSyntaxEntry
	if err := json.Unmarshal(filterSyntaxData, &filterSyntaxEntries); err != nil {
		// Log and continue if filter_syntax.json is invalid.
		// The server can start, but filter help will be unavailable or incomplete.
		log.Printf("WARNING: Error unmarshalling filter_syntax.json at %s: %v. Filter syntax help may be incomplete or unavailable.", filterSyntaxJSONPath, err)
		cfg.FilterSyntax = []FilterSyntaxEntry{} // Ensure it's empty
		return cfg, nil
	}

	cfg.FilterSyntax = filterSyntaxEntries
	log.Printf("Successfully loaded filter syntax help from %s. Found %d entries.", filterSyntaxJSONPath, len(cfg.FilterSyntax))
	return cfg, nil
}

// LoadDefaultConfiguration parses the default configuration data.
// The actual embedding of default.json should be done by the calling package (e.g., main).
func LoadDefaultConfiguration(defaultConfigData []byte) (Configuration, error) {
	// Note: This function currently only loads the default main configuration.
	// It does not attempt to load a default filter_syntax.json.
	// If default filter syntax were needed, similar logic to LoadConfig
	// would be required here, perhaps embedding a default filter_syntax.json.
	if defaultConfigData == nil {
		log.Println("Warning: No default configuration data provided to LoadDefaultConfiguration.")
		return Configuration{}, fmt.Errorf("default configuration data is nil")
	}
	return parseConfiguration(defaultConfigData)
}
