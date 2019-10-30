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
	APIKey              string `json:"api_key"`
	CustomID            string `json:"custom_id"`
	Server              string `json:"server"`
	ReportIntervalMin   int    `json:"report_interval_min"`
	PayloadCommand      string `json:"payload_command"`
	SSHEnabled          bool   `json:"ssh_enabled"`
	SSHLocalHost        string `json:"ssh_local_host"`
	SSHLocalPort        int    `json:"ssh_local_port"`
	SSHRetryGapSec      int    `json:"ssh_retry_gap_sec"`
	PingEnabled         bool   `json:"ping_enabled"`
	PrimaryPingTarget   string `json:"ping_primary"`
	SecondaryPingTarget string `json:"ping_secondary"`
}

var config = Config{
	ReportIntervalMin:   3,
	SSHLocalHost:        "localhost",
	SSHLocalPort:        22,
	SSHRetryGapSec:      10,
	PrimaryPingTarget:   "1.1.1.1",
	SecondaryPingTarget: "1.0.0.1",
}

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
	return nil
}

// SSHLocal returns SSH local host and port with colon separator.
func (c Config) SSHLocal() string {
	return fmt.Sprintf("%s:%d", c.SSHLocalHost, c.SSHLocalPort)
}
