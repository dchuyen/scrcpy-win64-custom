package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	deviceIP := flag.String("ip", "", "device Wi-Fi IP address; choose from saved IPs or enter manually")
	startPort := flag.Int("start", 30000, "start port")
	endPort := flag.Int("end", 45000, "end port")
	timeoutMs := flag.Int("timeout", 80, "timeout for each TCP probe in milliseconds")
	workers := flag.Int("workers", 500, "number of workers for concurrent scanning")
	flag.Parse()
	defer waitForExit()

	if *startPort <= 0 || *endPort <= 0 || *startPort > *endPort {
		fmt.Fprintln(os.Stderr, "invalid port range")
		os.Exit(2)
	}

	if *deviceIP == "" {
		savedIPs, _ := loadSavedDeviceIPs()
		if len(savedIPs) > 0 {
			*deviceIP = selectDeviceIP(savedIPs)
		} else {
			*deviceIP = promptForDeviceIP()
		}

		for *deviceIP == "" {
			fmt.Println("Please enter the device IP manually.")
			*deviceIP = promptForDeviceIP()
		}
		_ = saveDeviceIP(*deviceIP)
	} else {
		_ = saveDeviceIP(*deviceIP)
	}

	host, port, hasPort, err := parseDeviceTarget(*deviceIP)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid device target: %v\n", err)
		os.Exit(2)
	}

	if hasPort {
		address := fmt.Sprintf("%s:%d", host, port)
		scrcpyPath := resolveScrcpyPath()
		if scrcpyPath == "" {
			fmt.Println("scrcpy executable not found, skip launching")
			return
		}

		fmt.Printf("Launching scrcpy for %s...\n", address)
		if err := launchScrcpy(scrcpyPath, address); err != nil {
			fmt.Fprintf(os.Stderr, "failed to launch scrcpy: %v\n", err)
		}
		return
	}

	fmt.Printf("Scanning %s from port %d to %d with timeout=%dms...\n", host, *startPort, *endPort, *timeoutMs)

	openPorts, err := scanPorts(host, *startPort, *endPort, time.Duration(*timeoutMs)*time.Millisecond, *workers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan failed: %v\n", err)
		os.Exit(1)
	}

	if len(openPorts) == 0 {
		fmt.Println("No open ports found.")
		return
	}

	fmt.Println("Open ports:")
	for _, port := range openPorts {
		fmt.Printf("- %d\n", port)
	}

	selectedPort := openPorts[0]
	address := fmt.Sprintf("%s:%d", host, selectedPort)
	scrcpyPath := resolveScrcpyPath()
	if scrcpyPath == "" {
		fmt.Println("scrcpy executable not found, skip launching")
		return
	}

	fmt.Printf("Launching scrcpy for %s...\n", address)
	if err := launchScrcpy(scrcpyPath, address); err != nil {
		fmt.Fprintf(os.Stderr, "failed to launch scrcpy: %v\n", err)
	}
}
