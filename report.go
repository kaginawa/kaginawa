package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// report defines all of report attributes
type report struct {
	ID             string      `json:"id"`                         // MAC address of the primary network interface
	Trigger        int         `json:"trigger"`                    // Report trigger (-1: connected, 0: boot, n: timer)
	Runtime        string      `json:"runtime"`                    // OS and arch
	Success        bool        `json:"success"`                    // Equals len(Errors) == 0
	Sequence       int         `json:"seq"`                        // Report sequence number from process start
	DeviceTime     int64       `json:"device_time"`                // Device time (UTC) by time.Now().UTC().Unix()
	BootTime       int64       `json:"boot_time"`                  // Device boot time (UTC)
	GenMillis      int64       `json:"gen_ms"`                     // Generation time milliseconds
	AgentVersion   string      `json:"agent_version"`              // Agent version
	CustomID       string      `json:"custom_id,omitempty"`        // User specified ID
	SSHServerHost  string      `json:"ssh_server_host,omitempty"`  // Connected SSH server host
	SSHRemotePort  int         `json:"ssh_remote_port,omitempty"`  // Connected SSH remote port
	SSHConnectTime int64       `json:"ssh_connect_time,omitempty"` // Connected time of the SSH
	Adapter        string      `json:"adapter,omitempty"`          // Name of network adapter, source of the MAC address
	LocalIPv4      string      `json:"ip4_local,omitempty"`        // Local IPv6 address
	LocalIPv6      string      `json:"ip6_local,omitempty"`        // Local IPv6 address
	Hostname       string      `json:"hostname,omitempty"`         // OS Hostname
	RTTMills       int64       `json:"rtt_ms,omitempty"`           // Round trip time milliseconds
	UploadKBPS     int64       `json:"upload_bps,omitempty"`       // Upload throughput bps
	DownloadKBPS   int64       `json:"download_bps,omitempty"`     // Download throughput bps
	DiskTotalBytes int64       `json:"disk_total_bytes,omitempty"` // Total disk space (Bytes)
	DiskUsedBytes  int64       `json:"disk_used_bytes,omitempty"`  // Used disk space (Bytes)
	DiskLabel      string      `json:"disk_label,omitempty"`       // Disk label
	DiskFilesystem string      `json:"disk_filesystem,omitempty"`  // Disk filesystem name
	DiskMountPoint string      `json:"disk_mount_point,omitempty"` // Mount point (default is root)
	DiskDevice     string      `json:"disk_device,omitempty"`      // Disk device name
	USBDevices     []usbDevice `json:"usb_devices,omitempty"`      // List of usb devices
	BDLocalDevices []string    `json:"bd_local_devices,omitempty"` // List of Bluetooth local devices
	KernelVersion  string      `json:"kernel_version,omitempty"`   // Kernel version
	Errors         []string    `json:"errors,omitempty"`           // List of errors
	Payload        string      `json:"payload,omitempty"`          // Custom content provided by payload command
	PayloadCmd     string      `json:"payload_cmd,omitempty"`      // Executed payload command
}

type usbDevice struct {
	Name      string `json:"name,omitempty"`
	VendorID  string `json:"vendor_id,omitempty"`
	ProductID string `json:"product_id,omitempty"`
	Location  string `json:"location,omitempty"`
}

// reply defines all of reply message attributes
type reply struct {
	Reboot        bool   `json:"reboot,omitempty"` // Reboot requested from the server
	SSHServerHost string `json:"ssh_host,omitempty"`
	SSHServerPort int    `json:"ssh_port,omitempty"`
	SSHServerUser string `json:"ssh_user,omitempty"`
	SSHKey        string `json:"ssh_key,omitempty"`
	SSHPassword   string `json:"ssh_password,omitempty"`
}

var seq = 0

// doReport generates and uploads a record.
func doReport(trigger int) {
	if err := initID(); err != nil {
		log.Printf("failed to rescan ID: %v", err)
	}
	report := genReport(trigger)
	var data []byte
	var err error
	if *debugPrint {
		data, err = json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Fatalf("failed to marshal report: %v", err)
		}
		log.Printf("REPORT: %s", string(data))
	} else {
		data, err = json.Marshal(report)
		if err != nil {
			log.Fatalf("failed to marshal report: %v", err)
		}
	}
	if strings.Contains(config.Server, "localhost") {
		if err := uploadReport(data, "http"); err != nil {
			log.Printf("failed to upload report: %v", err)
		}
	} else {
		if err := uploadReport(data, "https"); err != nil {
			if strings.HasPrefix(err.Error(), "failed to upload with https:") {
				if err := uploadReport(data, "http"); err != nil {
					log.Printf("failed to upload report: %v", err)
				}
			} else {
				log.Printf("failed to upload report: %v", err)
			}
		}
	}
}

// genReport generates a report.
func genReport(trigger int) report {
	seq++
	timeBegin := time.Now()
	report := report{
		ID:             macAddr,
		Trigger:        trigger,
		CustomID:       config.CustomID,
		BootTime:       bootTime.Unix(),
		SSHServerHost:  msg.SSHServerHost,
		SSHRemotePort:  sshRemotePort,
		SSHConnectTime: sshConnectTime.Unix(),
		Sequence:       seq,
		Adapter:        adapterName,
		LocalIPv6:      localIPv6,
		LocalIPv4:      localIPv4,
		Runtime:        runtime.GOOS + " " + runtime.GOARCH,
		AgentVersion:   ver,
		KernelVersion:  kernelVersion(),
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to collect hostname: %v", err))
	}
	report.Hostname = hostname

	// Platform information
	if config.DiskUsageEnabled {
		if rep, err := diskUsage(config.DiskUsageMountPoint); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to obtain disk usage: %v", err))
		} else {
			report.DiskTotalBytes = rep.TotalBytes
			report.DiskUsedBytes = rep.UsedBytes
			report.DiskLabel = rep.Label
			report.DiskFilesystem = rep.Filesystem
			report.DiskMountPoint = rep.MountPoint
			report.DiskDevice = rep.Device
		}
	}
	if config.USBScanEnabled {
		if rep, err := usbDevices(); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to obtain list of usb devices: %v", err))
		} else {
			report.USBDevices = rep
		}
	}
	if config.BTScanEnabled {
		if rep, err := bdLocalDevices(); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to obtain list of bluetooth devices: %v", err))
		} else {
			report.BDLocalDevices = rep
		}
	}

	// Measurements
	if config.RTTEnabled {
		if rtt, err := measureRoundTripTimeMills(); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to measure rtt: %v", err))
		} else {
			report.RTTMills = rtt
		}
	}
	if config.ThroughputEnabled && config.ThroughputKB >= 0 {
		if downKBPS, upKBPS, err := measureThroughput(config.ThroughputKB); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to measure throughput: %v", err))
		} else {
			report.DownloadKBPS = downKBPS
			report.UploadKBPS = upKBPS
		}
	}

	// Payload
	if len(config.PayloadCommand) > 0 {
		report.PayloadCmd = config.PayloadCommand
		param := strings.Split(config.PayloadCommand, " ")
		name := param[0]
		args := make([]string, 0)
		if len(param) > 1 {
			args = param[1:]
		}
		output, err := exec.Command(name, args...).Output()
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to execute payload command: %v", err))
		}
		if output != nil {
			report.Payload = string(output)
		}
	}

	// Final status
	report.Success = len(report.Errors) == 0
	report.DeviceTime = time.Now().UTC().Unix()
	report.GenMillis = time.Since(timeBegin).Milliseconds()

	return report
}

// uploadReport uploads a report with specified proto (http or https).
func uploadReport(report []byte, proto string) error {
	gz := new(bytes.Buffer)
	w := gzip.NewWriter(gz)
	if _, err := w.Write(report); err != nil {
		return fmt.Errorf("failed to gzip report: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close gzipped report: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, proto+"://"+config.Server+"/report", gz)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+config.APIKey)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload with %s: %v", proto, err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	reader := resp.Body
	defer safeClose(reader, "report body")
	if resp.Header.Get("Content-Encoding") == "gzip" {
		r, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read gzipped response: %w", err)
		}
		reader = r
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("faield to read response: %w", err)
	}
	var serverMessage reply
	if err := json.Unmarshal(body, &serverMessage); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Start listening SSH if not started
	if config.SSHEnabled {
		msg = serverMessage
		sshLoopStarted.Do(func() { go listenSSH() })
	}
	return nil
}

// SSHServer returns SSH server host and port with colon separator.
func (r reply) SSHServer() string {
	return fmt.Sprintf("%s:%d", r.SSHServerHost, r.SSHServerPort)
}
