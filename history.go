package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func historyFilePath() string {
	wd, err := os.Getwd()
	if err != nil {
		return "device_ips.txt"
	}
	return filepath.Join(wd, "device_ips.txt")
}

func parseDeviceTarget(target string) (string, int, bool, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", 0, false, fmt.Errorf("empty target")
	}

	if ip := net.ParseIP(target); ip != nil {
		return target, 0, false, nil
	}

	host, portText, err := net.SplitHostPort(target)
	if err != nil {
		return "", 0, false, fmt.Errorf("invalid target %q: %w", target, err)
	}
	if net.ParseIP(host) == nil {
		return "", 0, false, fmt.Errorf("invalid host %q", host)
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, false, fmt.Errorf("invalid port %q", portText)
	}
	return host, port, true, nil
}

func loadSavedDeviceIPs() ([]string, error) {
	path := historyFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var ips []string
	seen := make(map[string]bool)
	for _, line := range lines {
		ip := strings.TrimSpace(line)
		if ip == "" || seen[ip] {
			continue
		}
		if _, _, _, err := parseDeviceTarget(ip); err != nil {
			continue
		}
		seen[ip] = true
		ips = append(ips, ip)
	}
	return ips, nil
}

func saveDeviceIP(target string) error {
	if _, _, _, err := parseDeviceTarget(target); err != nil {
		return nil
	}

	ips, err := loadSavedDeviceIPs()
	if err != nil {
		return err
	}
	for _, existing := range ips {
		if existing == target {
			return nil
		}
	}
	f, err := os.OpenFile(historyFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, target)
	return err
}

func promptForDeviceIP() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter device IP or IP:port: ")
	ip, _ := reader.ReadString('\n')
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if _, _, _, err := parseDeviceTarget(ip); err != nil {
		fmt.Fprintf(os.Stderr, "invalid device target: %s\n", ip)
		return promptForDeviceIP()
	}
	return ip
}

func selectDeviceIP(candidates []string) string {
	if len(candidates) == 0 {
		return promptForDeviceIP()
	}

	fmt.Println("Available device IPs:")
	for i, ip := range candidates {
		fmt.Printf("%d) %s\n", i+1, ip)
	}
	fmt.Println("Enter a number to choose an existing IP, or type a new IP directly.")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Select an IP or enter a new IP: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		if choice == "" {
			fmt.Println("Please choose a number or enter a new IP.")
			continue
		}
		if _, _, _, err := parseDeviceTarget(choice); err == nil {
			return choice
		}
		index, err := strconv.Atoi(choice)
		if err != nil || index < 1 || index > len(candidates) {
			fmt.Fprintln(os.Stderr, "Invalid selection. Please choose a valid number or enter a new IP.")
			continue
		}
		return candidates[index-1]
	}
}

func waitForExit() {
	fmt.Print("Press Enter to exit...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
