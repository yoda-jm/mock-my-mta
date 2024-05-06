package main

import (
	_ "embed"
	"encoding/json"
	"mock-my-mta/storage"
	"os"
)

// embedded default configuration
//
//go:embed config/default.json
var defaultConfigurationData []byte

type Configuration struct {
	Smtpd    SmtpdConfiguration                  `json:"smtpd"`
	Httpd    HttpdConfiguration                  `json:"httpd"`
	Storages []storage.StorageLayerConfiguration `json:"storages"`
	Logging  LoggingConfiguration                `json:"logging"`
}

type SmtpdConfiguration struct {
	Addr      string `json:"addr"`
	RelayAddr string `json:"relay-addr"`
}

type HttpdConfiguration struct {
	Addr string `json:"addr"`
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
