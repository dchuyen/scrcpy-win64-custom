# scrcpy-win64-custom (Port Scanner + Launcher)

A small Windows utility that locates an Android device listening for scrcpy over TCP, optionally scans a configurable port range to find an open scrcpy port, and launches scrcpy to mirror the device. It saves previously used device targets and provides an interactive prompt when no IP is supplied.

## Features
- Accepts device target as IP or IP:port; if a port is included the tool launches scrcpy directly.
- Concurrent TCP port scanner with configurable port range, timeout, and worker count.
- Saves and lets you choose from previously used device targets in device_ips.txt.
- Attempts to locate scrcpy executable in the current directory or common scrcpy-win64 folder names.
- Simple interactive prompts for selecting or entering device targets.
- Can be built from source (Go 1.20+) or used with the included prebuilt binary.

## Requirements
- Windows (tool resolves `scrcpy.exe` and common scrcpy-win64 directories).
- scrcpy for Windows (place scrcpy.exe in the working directory or in a scrcpy-win64-v3.3.4 folder adjacent to the binary).
- Go 1.20+ (only needed to build from source).
- The target Android device must have adb TCP enabled (e.g., `adb tcpip 5555`) or otherwise be listening for scrcpy connections.

## Security note
A prebuilt `portscan.exe` may be present in the repository. Verify the binary before running it. If unsure, build from source.

## Build
From the repository root:
```
go build -o portscan.exe
```

## Usage
Basic usage examples:
- Connect using a saved or entered IP:
  ```
  portscan.exe -ip 192.168.1.100
  ```
- Connect if you already know the port:
  ```
  portscan.exe -ip 192.168.1.100:5555
  ```
- Scan a port range:
  ```
  portscan.exe -ip 192.168.1.100 -start 30000 -end 45000 -timeout 80 -workers 500
  ```

Command-line flags:
- `-ip string` — device Wi-Fi IP address; choose from saved targets or enter manually (default: "")
- `-start int` — start port (default: 30000)
- `-end int` — end port (default: 45000)
- `-timeout int` — timeout for each TCP probe in milliseconds (default: 80)
- `-workers int` — number of concurrent workers for scanning (default: 500)

## How it works (brief)
1. If `-ip` is not provided, the tool reads `device_ips.txt` (in the working directory) and prompts you to pick an existing target or enter a new one.
2. If the provided target includes a port (IP:port), it verifies scrcpy is available and invokes:
   ```
   scrcpy --tcpip=host:port
   ```
3. Otherwise it concurrently scans the configured port range for open TCP ports on the device and prints any found open ports.
4. The first open port found is chosen and scrcpy is launched against it.
5. Successful targets are appended to `device_ips.txt` for future use.

## Example workflow
1. Enable adb TCP on the device (via USB or device shell):
   ```
   adb tcpip 5555
   ```
2. Run the scanner/launcher:
   ```
   portscan.exe -ip 192.168.1.42
   ```
3. If scanning, wait for the tool to list open ports, then it will launch scrcpy for the chosen open port.

## Files of interest
- main.go — entry point; command-line parsing and overall flow.
- network.go — concurrent port scanner and TCP probe logic.
- history.go — device target history (load/save/select/prompt helpers).
- adb.go — scrcpy path resolution and launcher helper.
- device_ips.txt — saved device targets (created/updated by the tool).
- portscan.exe — optional prebuilt Windows binary (if present; verify before running).

## Troubleshooting
- scrcpy not found: ensure `scrcpy.exe` exists in the current working directory or in a `scrcpy-win64-v3.3.4` folder adjacent to the executable.
- No open ports found: confirm the device is listening for TCP (use `adb tcpip <port>` on the device).
- Permission or binary issues: prefer building from source with Go if you cannot verify a prebuilt binary.

## Contributing
Contributions, bug reports, and feature requests are welcome. Open an issue or submit a pull request.

## License
No license file is included in this repository (if you want a license, add one appropriate to your project).
