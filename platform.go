package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"text/scanner"
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
	StorageDataType []darwinStorageDataType `json:"SPStorageDataType"`
}

type darwinStorageDataType struct {
	Name       string `json:"_name"`
	BSDName    string `json:"bsd_name"`
	Filesystem string `json:"file_system"`
	FreeBytes  int64  `json:"free_space_in_bytes"`
	MountPoint string `json:"mount_point"`
	TotalBytes int64  `json:"size_in_bytes"`
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
		for _, address := range addresses {
			if len(localIPv6) == 0 && strings.Contains(address.String(), ":") {
				localIPv6 = trimSubnetMusk(address)
			} else if len(localIPv4) == 0 && strings.Contains(address.String(), ".") {
				localIPv4 = trimSubnetMusk(address)
			}
		}
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
		for _, record := range profile.StorageDataType {
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
		raw, err := exec.Command("df", "-T", mountPoint).Output()
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(raw), "\n")
		if len(lines) < 2 {
			return nil, fmt.Errorf("no record: %s", string(raw))
		}
		var s scanner.Scanner
		s.Init(strings.NewReader(lines[1]))
		tokens := make([]string, 0)
		for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
			tokens = append(tokens, s.TokenText())
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

func trimSubnetMusk(addr net.Addr) string {
	i := strings.Index(addr.String(), "/")
	if i > 0 {
		return addr.String()[0:i]
	}
	return addr.String()
}
