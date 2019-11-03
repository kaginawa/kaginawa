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

const defaultConfigFilePath = "kaginawa.json"

var (
	ver            = "v0.0.0"
	configPath     = flag.String("c", defaultConfigFilePath, "path to configuration file")
	versionPrint   = flag.Bool("v", false, "print version and exit")
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
	if err := initID(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Kaginawa %s on %s", ver, macAddr)

	// Update checker
	if config.UpdateEnabled {
		go updateChecker()
	}

	// Main loop
	doReport()
	for range time.Tick(time.Duration(config.ReportIntervalMin) * time.Minute) {
		doReport()
	}
}

func safeClose(closer io.Closer, name string) {
	if err := closer.Close(); err != nil {
		log.Printf("failed to close %s: %v", name, err)
	}
}
