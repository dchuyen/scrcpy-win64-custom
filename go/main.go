package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	// Định nghĩa các tham số dòng lệnh: đường dẫn ADB, IP thiết bị, phạm vi cổng, timeout và số worker.
	adbPathFlag := flag.String("adb", "", "path to adb executable")
	deviceIP := flag.String("ip", "", "device Wi-Fi IP address; if empty, try to get it from adb")
	startPort := flag.Int("start", 30000, "start port")
	endPort := flag.Int("end", 45000, "end port")
	timeoutMs := flag.Int("timeout", 80, "timeout for each TCP probe in milliseconds")
	workers := flag.Int("workers", 500, "number of workers for concurrent scanning")
	flag.Parse()
	defer waitForExit()

	// Kiểm tra phạm vi cổng nhập vào có hợp lệ không.
	if *startPort <= 0 || *endPort <= 0 || *startPort > *endPort {
		fmt.Fprintln(os.Stderr, "invalid port range")
		os.Exit(2)
	}

	// Tìm đường dẫn tới file adb để có thể giao tiếp với thiết bị Android.
	adbPath := resolveADBPath(*adbPathFlag)

	// Nếu chưa có IP thì chọn từ lịch sử hoặc hỏi người dùng.
	if *deviceIP == "" {
		savedIPs, _ := loadSavedDeviceIPs()
		*deviceIP = selectDeviceIP(savedIPs)
		if *deviceIP == "" {
			resolvedIP, err := resolveDeviceIP(adbPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cannot resolve device IP: %v\n", err)
				os.Exit(1)
			}
			*deviceIP = resolvedIP
		}
		_ = saveDeviceIP(*deviceIP)
	} else {
		_ = saveDeviceIP(*deviceIP)
	}

	fmt.Printf("Scanning %s from port %d to %d with timeout=%dms...\n", *deviceIP, *startPort, *endPort, *timeoutMs)

	// Quét các cổng TCP trên IP thiết bị và lấy danh sách cổng đang mở.
	openPorts, err := scanPorts(*deviceIP, *startPort, *endPort, time.Duration(*timeoutMs)*time.Millisecond, *workers)
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

	port := openPorts[0]
	address := fmt.Sprintf("%s:%d", *deviceIP, port)
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

func resolveADBPath(explicit string) string {
	// Nếu người dùng đã chỉ định đường dẫn adb thì dùng ngay.
	if explicit != "" {
		return explicit
	}

	// Nếu không, thử các vị trí phổ biến trên Windows.
	candidates := []string{"adb", ".\\adb.exe", ".\\adb", ".\\platform-tools\\adb.exe", ".\\platform-tools\\adb"}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "adb"
}

func resolveScrcpyPath() string {
	baseDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	parentDir := filepath.Dir(baseDir)

	candidates := []string{
		filepath.Join(baseDir, "scrcpy.exe"),
		filepath.Join(baseDir, "scrcpy-win64-v3.3.4", "scrcpy.exe"),
		filepath.Join(parentDir, "scrcpy-win64-v3.3.4", "scrcpy.exe"),
		filepath.Join(parentDir, "scrcpy-win64-v3.3.4", "scrcpy-console.bat"),
		filepath.Join(parentDir, "scrcpy-win64-v3.3.4", "scrcpy-noconsole.vbs"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func launchScrcpy(scrcpyPath, target string) error {
	cmd := exec.Command(scrcpyPath, "--tcpip="+target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Dir(scrcpyPath)
	return cmd.Start()
}

func promptForDeviceIP() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter device IP (leave blank to auto-detect): ")
	ip, _ := reader.ReadString('\n')
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if net.ParseIP(ip) == nil {
		fmt.Fprintf(os.Stderr, "invalid IP address: %s\n", ip)
		return promptForDeviceIP()
	}
	return ip
}

func selectDeviceIP(savedIPs []string) string {
	if len(savedIPs) == 0 {
		return promptForDeviceIP()
	}

	fmt.Println("Saved device IPs:")
	for i, ip := range savedIPs {
		fmt.Printf("%d) %s\n", i+1, ip)
	}
	fmt.Printf("0) Enter a new IP\n")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Select an IP or enter 0: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		if choice == "0" || choice == "" {
			return promptForDeviceIP()
		}
		index, err := strconv.Atoi(choice)
		if err != nil || index < 1 || index > len(savedIPs) {
			fmt.Fprintln(os.Stderr, "Invalid selection. Please choose a valid number.")
			continue
		}
		return savedIPs[index-1]
	}
}

func historyFilePath() string {
	wd, err := os.Getwd()
	if err != nil {
		return "device_ips.txt"
	}
	return filepath.Join(wd, "device_ips.txt")
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
		if ip == "" || seen[ip] || net.ParseIP(ip) == nil {
			continue
		}
		seen[ip] = true
		ips = append(ips, ip)
	}
	return ips, nil
}

func saveDeviceIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return nil
	}

	ips, err := loadSavedDeviceIPs()
	if err != nil {
		return err
	}
	for _, existing := range ips {
		if existing == ip {
			return nil
		}
	}
	f, err := os.OpenFile(historyFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, ip)
	return err
}

func resolveDeviceIP(adbPath string) (string, error) {
	// Gọi adb để đọc IP Wi-Fi của thiết bị từ bên trong Android.
	for _, args := range [][]string{
		{"shell", "ip", "-f", "inet", "addr", "show", "wlan0"},
		{"shell", "ip", "route"},
	} {
		out, err := runADB(adbPath, args)
		if err != nil {
			continue
		}

		if ip := parseIPFromADBOutput(out); ip != "" {
			return ip, nil
		}
	}

	return "", fmt.Errorf("could not infer Wi-Fi IP from adb output")
}

func parseIPFromADBOutput(output string) string {
	// Dùng regex để lấy địa chỉ IP từ output của lệnh adb.
	re := regexp.MustCompile(`(?:inet|src)\s+(\d{1,3}(?:\.\d{1,3}){3})`)
	matches := re.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		if m[1] != "" && m[1] != "127.0.0.1" {
			return m[1]
		}
	}
	return ""
}

func runADB(adbPath string, args []string) (string, error) {
	// Chạy lệnh adb với timeout ngắn để tránh treo khi thiết bị không phản hồi.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, adbPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("adb %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}

	return strings.TrimSpace(string(out)), nil
}

func scanPorts(host string, startPort, endPort int, timeout time.Duration, workers int) ([]int, error) {
	// Nếu số worker không hợp lệ thì dùng giá trị mặc định.
	if workers <= 0 {
		workers = 200
	}

	// Tạo hàng đợi công việc và kết quả để quét song song nhiều cổng cùng lúc.
	jobs := make(chan int)
	results := make(chan int, 1000)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range jobs {
				if isOpen(host, port, timeout) {
					results <- port
				}
			}
		}()
	}

	go func() {
		for port := startPort; port <= endPort; port++ {
			jobs <- port
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var open []int
	for port := range results {
		open = append(open, port)
	}

	sort.Ints(open)
	return open, nil
}

func isOpen(host string, port int, timeout time.Duration) bool {
	// Thử kết nối TCP tới một cổng cụ thể; nếu thành công thì cổng đang mở.
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func waitForExit() {
	fmt.Print("Press Enter to exit...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
