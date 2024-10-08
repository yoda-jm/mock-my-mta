package main

import (
	_ "embed"
	"encoding/json"
	"mock-my-mta/http"
	"mock-my-mta/smtp"
	"mock-my-mta/storage"
	"os"
)

// embedded default configuration
//
//go:embed config/default.json
var defaultConfigurationData []byte

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

func readConfigurationFile(filename string) (Configuration, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Configuration{}, err
	}
	return parseConfiguration(data)
}

func loadDefaultConfiguration() (Configuration, error) {
	return parseConfiguration(defaultConfigurationData)
}
