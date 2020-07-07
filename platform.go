package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type diskUsageReport struct {
	TotalBytes int64
	UsedBytes  int64
	Label      string
	Filesystem string
	MountPoint string
	Device     string
}

type darwinSystemProfile struct {
	Storage   []darwinStorageDataType   `json:"SPStorageDataType"`
	USB       []darwinUSBDataType       `json:"SPUSBDataType"`
	Bluetooth []darwinBluetoothDataType `json:"SPBluetoothDataType"`
}

type darwinStorageDataType struct {
	Name       string `json:"_name"`
	BSDName    string `json:"bsd_name"`
	Filesystem string `json:"file_system"`
	FreeBytes  int64  `json:"free_space_in_bytes"`
	MountPoint string `json:"mount_point"`
	TotalBytes int64  `json:"size_in_bytes"`
}

type darwinUSBDataType struct {
	Name      string              `json:"_name"`
	VendorID  string              `json:"vendor_id"`
	ProductID string              `json:"product_id"`
	Location  string              `json:"location_id"`
	Items     []darwinUSBDataType `json:"_items"`
}

type darwinBluetoothDataType struct {
	LocalDeviceTitle struct {
		Address string `json:"general_address"`
	} `json:"local_device_title"`
}

func initID() error {
	adapters, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to collect network interface: %w", err)
	}
	for _, adapter := range adapters {
		if adapter.HardwareAddr == nil || !strings.Contains(adapter.Flags.String(), "up") {
			continue // ignore odd or down adapters
		}
		addresses, err := adapter.Addrs()
		if err != nil || len(addresses) == 0 {
			continue // ignore unassigned adapters
		}
		var v4, v6 string
		for _, address := range addresses {
			if len(v6) == 0 && strings.Contains(address.String(), ":") {
				v6 = trimSubnetMusk(address)
			}
			if len(v4) == 0 && strings.Contains(address.String(), ".") {
				v4 = trimSubnetMusk(address)
			}
		}
		localIPv4 = v4
		localIPv6 = v6
		macAddr = strings.ToLower(adapter.HardwareAddr.String())
		adapterName = adapter.Name
		break
	}
	if len(macAddr) == 0 {
		return errors.New("no adapter available")
	}
	return nil
}

func diskUsage(mountPoint string) (*diskUsageReport, error) {
	switch runtime.GOOS {
	case "darwin":
		raw, err := exec.Command("system_profiler", "-json", "SPStorageDataType").Output()
		if err != nil {
			return nil, err
		}
		var profile darwinSystemProfile
		if err := json.Unmarshal(raw, &profile); err != nil {
			return nil, err
		}
		for _, record := range profile.Storage {
			if record.MountPoint == mountPoint {
				return &diskUsageReport{
					TotalBytes: record.TotalBytes,
					UsedBytes:  record.TotalBytes - record.FreeBytes,
					Label:      record.Name,
					Filesystem: record.Filesystem,
					MountPoint: record.MountPoint,
					Device:     "/dev/" + record.BSDName,
				}, nil
			}
		}
		return nil, fmt.Errorf("no storage profile: %s", string(raw))
	case "linux":
		raw, err := exec.Command("df", "-T", "-B", "1", mountPoint).Output()
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(raw), "\n")
		if len(lines) < 2 {
			return nil, fmt.Errorf("no record: %s", string(raw))
		}
		tokens := make([]string, 0)
		for _, tok := range strings.Split(lines[1], " ") {
			if len(tok) > 0 {
				tokens = append(tokens, tok)
			}
		}
		if len(tokens) < 7 {
			return nil, fmt.Errorf("invalid record: %s", lines[1])
		}
		used, err := strconv.ParseInt(tokens[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid record: %s", lines[1])
		}
		available, err := strconv.ParseInt(tokens[4], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid record: %s", lines[1])
		}
		return &diskUsageReport{
			TotalBytes: used + available,
			UsedBytes:  used,
			Filesystem: tokens[1],
			MountPoint: tokens[6],
			Device:     tokens[0],
		}, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func usbDevices() ([]usbDevice, error) {
	switch runtime.GOOS {
	case "darwin":
		raw, err := exec.Command("system_profiler", "-json", "SPUSBDataType").Output()
		if err != nil {
			return nil, err
		}
		var profile darwinSystemProfile
		if err := json.Unmarshal(raw, &profile); err != nil {
			return nil, err
		}
		return extractUSBProfile(profile.USB), nil
	case "linux":
		raw, err := exec.Command("lsusb").Output()
		if err != nil {
			return nil, err
		}
		var devices []usbDevice
		for _, line := range strings.Split(string(raw), "\n") {
			tokens := strings.Split(line, " ")
			if len(tokens) < 7 {
				continue
			}
			devices = append(devices, usbDevice{
				Name:      strings.Join(tokens[6:], " "),
				VendorID:  tokens[5][0:4],
				ProductID: tokens[5][5:9],
				Location:  strings.TrimRight(strings.Join(tokens[0:4], " "), ":"),
			})
		}
		return devices, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func bdLocalDevices() ([]string, error) {
	switch runtime.GOOS {
	case "darwin":
		raw, err := exec.Command("system_profiler", "-json", "SPBluetoothDataType").Output()
		if err != nil {
			return nil, err
		}
		var profile darwinSystemProfile
		if err := json.Unmarshal(raw, &profile); err != nil {
			return nil, err
		}
		var addresses []string
		for _, item := range profile.Bluetooth {
			if len(item.LocalDeviceTitle.Address) > 0 {
				addresses = append(addresses, strings.ReplaceAll(item.LocalDeviceTitle.Address, "-", ":"))
			}
		}
		return addresses, nil
	case "linux":
		raw, err := exec.Command("hcitool", "dev").Output()
		if err != nil {
			return nil, err
		}
		var addresses []string
		for _, line := range strings.Split(string(raw), "\n") {
			tokens := strings.Split(line, "\t")
			if len(tokens) != 3 {
				continue
			}
			addresses = append(addresses, tokens[2])
		}
		return addresses, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func trimSubnetMusk(addr net.Addr) string {
	i := strings.Index(addr.String(), "/")
	if i > 0 {
		return addr.String()[0:i]
	}
	return addr.String()
}

func extractUSBProfile(list []darwinUSBDataType) []usbDevice {
	var devices []usbDevice
	for _, item := range list {
		if strings.HasPrefix(item.VendorID, "0x") && len(item.VendorID) >= 6 && len(item.ProductID) >= 6 {
			devices = append(devices, usbDevice{
				Name:      strings.Trim(item.VendorID[6:], " ()") + " " + item.Name,
				VendorID:  item.VendorID[2:6],
				ProductID: item.ProductID[2:6],
				Location:  item.Location,
			})
		}
		if len(item.Items) > 0 {
			devices = append(devices, extractUSBProfile(item.Items)...)
		}
	}
	return devices
}

func kernelVersion() string {
	if runtime.GOOS == "windows" {
		v, err := exec.Command("systeminfo", "/FO", "CSV").Output()
		if err != nil {
			log.Printf("failed to execute systeminfo: %v", err)
			return ""
		}
		records, err := csv.NewReader(bytes.NewReader(v)).ReadAll()
		if err != nil {
			log.Printf("failed to parse systeminfo: %v", err)
			return ""
		}
		if len(records) < 2 {
			log.Printf("systeminfo too short: len(records) = %d", len(records))
			return ""
		}
		record := records[1]
		if len(record) < 3 {
			log.Printf("systeminfo too short: len(record) = %d", len(record))
			return ""
		}
		return record[2]
	}
	v, err := exec.Command("uname", "-r").Output()
	if err != nil {
		log.Printf("failed to execute uname -r: %v", err)
		return ""
	}
	return strings.TrimRight(string(v), "\n")
}
