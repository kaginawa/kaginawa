package main

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

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
