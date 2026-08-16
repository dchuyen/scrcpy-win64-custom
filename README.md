Project description

A small Windows utility that locates an Android device listening for scrcpy over TCP, optionally scans a configurable port range to find an open scrcpy port, and launches scrcpy to mirror the device. It saves previously used device IPs and provides a simple interactive prompt when no IP is supplied.

Key features
- Accepts device target as IP or IP:port; if a port is included the tool launches scrcpy directly.
- Concurrent TCP port scanner with configurable port range, timeout, and worker count.
- Automatically saves and lets you choose from previously used device IPs (device_ips.txt).
- Attempts to locate scrcpy executable in the current directory or common scrcpy-win64 folder names.
- Can be built from source (Go 1.20+) or used with the included prebuilt binary.

Usage
- Basic: portscan.exe -ip 192.168.1.100
- If you already know the TCP port: portscan.exe -ip 192.168.1.100:5555
- Scanning example: portscan.exe -ip 192.168.1.100 -start 30000 -end 45000 -timeout 80 -workers 500

Command-line flags
- -ip string — device Wi-Fi IP address; choose from saved IPs or enter manually (default: "")
- -start int — start port (default: 30000)
- -end int — end port (default: 45000)
- -timeout int — timeout for each TCP probe in milliseconds (default: 80)
- -workers int — number of concurrent workers for scanning (default: 500)

How it works (brief)
1. If no -ip is provided, the tool reads device_ips.txt and prompts you to pick an IP or enter one.
2. If the provided target includes a port, it verifies scrcpy is available and invokes scrcpy --tcpip=host:port.
3. Otherwise it concurrently scans the configured port range for open TCP ports on the device.
4. If open ports are found, the first open port is chosen and scrcpy is launched against it.
5. Successful targets are appended to device_ips.txt for future use.

Requirements & notes
- Windows (the resolver searches for scrcpy.exe and scrcpy-win64-v3.3.4 in the current or parent directory).
- Ensure the device is accepting TCP connections for scrcpy (you may need to enable adb tcpip on the device beforehand).
- Building from source requires Go 1.20 or later.
- The repository contains a prebuilt portscan.exe (verify before running any binaries).

Build
- go build -o portscan.exe

Example workflow
- Connect your device via USB, run adb tcpip 5555 (or enable the device to listen on a port), then use this tool to find and connect:
  1) portscan.exe -ip 192.168.1.42
  2) Select or confirm an IP (or let the scanner find the port)
  3) The tool launches scrcpy pointing to the discovered host:port

If you want, I can convert this into a full README.md file for your repository.
