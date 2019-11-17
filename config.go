package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
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
	RTTEnabled          bool   `json:"rtt_enabled"`
	ThroughputEnabled   bool   `json:"throughput_enabled"`
	ThroughputKB        int    `json:"throughput_kb"`
	DiskUsageEnabled    bool   `json:"disk_usage_enabled"`
	DiskUsageMountPoint string `json:"disk_usage_mount_point"`
	USBScanEnabled      bool   `json:"usb_scan_enabled"`
	BTScanEnabled       bool   `json:"bt_scan_enabled"`
	UpdateEnabled       bool   `json:"update_enabled"`
	UpdateCheckURL      string `json:"update_check_url"`
	UpdateCommand       string `json:"update_command"`
}

var config = Config{
	ReportIntervalMin:   3,
	SSHEnabled:          true,
	SSHLocalHost:        "localhost",
	SSHLocalPort:        22,
	SSHRetryGapSec:      10,
	RTTEnabled:          true,
	ThroughputKB:        500,
	DiskUsageMountPoint: "/",
	UpdateEnabled:       true,
	UpdateCheckURL:      "https://kaginawa.github.io/LATEST",
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

	// Set OS-specific default value
	switch runtime.GOOS {
	case "darwin":
		config.DiskUsageEnabled = true
	case "linux":
		config.UpdateCommand = "sudo service kaginawa restart"
		config.DiskUsageEnabled = true
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
