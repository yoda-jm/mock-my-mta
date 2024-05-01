package main

import (
	"encoding/json"
	"mock-my-mta/storage"
)

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
