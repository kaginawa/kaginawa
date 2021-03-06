package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"runtime"
	"sync"
	"time"
)

const (
	defaultConfigFilePath = "kaginawa.json"
	macDetectionRetrySec  = 15
)

var (
	ver            = "v0.0.0"
	configPath     = flag.String("c", defaultConfigFilePath, "path to configuration file")
	versionPrint   = flag.Bool("v", false, "print version and exit")
	debugPrint     = flag.Bool("d", false, "log report content")
	bootTime       time.Time
	macAddr        string
	adapterName    string
	localIPv6      string
	localIPv4      string
	sshLoopStarted sync.Once
	sshConnectTime time.Time
	sshRemotePort  = 0
)

func main() {
	bootTime = time.Now().UTC()
	flag.Parse()

	// Print version
	if *versionPrint {
		fmt.Printf("kaginawa %s, compiled by %s\n", ver, runtime.Version())
		return
	}

	// Load configuration
	if err := loadConfig(*configPath); err != nil {
		log.Fatal(err)
	}

	// Determine the ID
	for {
		if err := initID(); err != nil {
			fmt.Printf("failed to determine active network interface: %v\n", err)
			fmt.Printf("retring after %d sec...\n", macDetectionRetrySec)
			time.Sleep(macDetectionRetrySec * time.Second)
			continue
		}
		break
	}
	log.Printf("Kaginawa %s on %s", ver, macAddr)

	// Update checker
	if config.UpdateEnabled {
		go updateChecker()
	}

	// Main loop
	doReport(0)
	for range time.Tick(time.Duration(config.ReportIntervalMin) * time.Minute) {
		doReport(config.ReportIntervalMin)
	}
}

func safeClose(closer io.Closer, name string) {
	if err := closer.Close(); err != nil {
		log.Printf("failed to close %s: %v", name, err)
	}
}
