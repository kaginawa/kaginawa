package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

// Config defines all of configuration parameters.
type Config struct {
	APIKey            string `json:"api_key"`
	CustomID          string `json:"custom_id"`
	Server            string `json:"server"`
	ReportIntervalMin int    `json:"report_interval_min"`
	PayloadCommand    string `json:"payload_command"`
	SSHEnabled        bool   `json:"ssh_enabled"`
	SSHLocalHost      string `json:"ssh_local_host"`
	SSHLocalPort      int    `json:"ssh_local_port"`
	SSHRetryGapSec    int    `json:"ssh_retry_gap_sec"`
}

var (
	config        Config
	defaultConfig = Config{
		ReportIntervalMin: 3,
		SSHLocalHost:      "localhost",
		SSHLocalPort:      22,
		SSHRetryGapSec:    10,
	}
)

// loadConfig loads configuration file from default or specified path.
func loadConfig(path string) error {
	if len(path) == 0 {
		path = defaultConfigFilePath
	}

	// Load file
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("configuration file not found: %s", path)
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", path, err)
	}

	// Parse file
	config = Config{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Validation
	if len(config.APIKey) == 0 {
		return errors.New("no api key configured")
	}
	if len(config.Server) == 0 {
		return errors.New("no server configured")
	}

	// Store default if zero value
	if config.ReportIntervalMin < 1 {
		config.ReportIntervalMin = defaultConfig.ReportIntervalMin
	}
	if len(config.SSHLocalHost) == 0 {
		config.SSHLocalHost = defaultConfig.SSHLocalHost
	}
	if config.SSHLocalPort < 1 {
		config.SSHLocalPort = defaultConfig.SSHLocalPort
	}
	if config.SSHRetryGapSec < 1 {
		config.SSHRetryGapSec = defaultConfig.SSHRetryGapSec
	}
	return nil
}

// SSHLocal returns SSH local host and port with colon separator.
func (c Config) SSHLocal() string {
	return fmt.Sprintf("%s:%d", c.SSHLocalHost, c.SSHLocalPort)
}
