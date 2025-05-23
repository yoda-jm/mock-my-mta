package main

import (
	_ "embed"
	"encoding/json"
	"os"

	"mock-my-mta/http"
	"mock-my-mta/smtp"
	"mock-my-mta/storage"
)

// embedded default configuration
//
//go:embed config/default.json
var defaultConfigurationData []byte

// Configuration holds the application configurations
type Configuration struct {
	Smtpd    smtp.Configuration                  `json:"smtpd"`
	Httpd    http.Configuration                  `json:"httpd"`
	Storages []storage.StorageLayerConfiguration `json:"storages"`
	Logging  LoggingConfiguration                `json:"logging"`
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

// LoadConfig reads the main configuration file
func LoadConfig(filename string) (Configuration, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Configuration{}, err
	}
	return parseConfiguration(data)
}

// LoadDefaultConfiguration parses the default configuration data.
func LoadDefaultConfiguration() (Configuration, error) {
	return parseConfiguration(defaultConfigurationData)
}
