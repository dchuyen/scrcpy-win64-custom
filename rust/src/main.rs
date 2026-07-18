use std::env;
use std::io;
use std::net::{SocketAddr, TcpStream};
use std::path::PathBuf;
use std::process::Command;
use std::process::Stdio;
use std::sync::mpsc;
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::Duration;
use regex::Regex;

fn main() {
    let mut args = env::args().skip(1);
    let mut adb_path = String::from("adb");
    let mut device_ip = String::new();
    let mut start_port = 30000u16;
    let mut end_port = 45000u16;
    let mut timeout_ms = 80u64;
    let mut workers = 500usize;

    while let Some(arg) = args.next() {
        match arg.as_str() {
            "-adb" => adb_path = args.next().unwrap_or_default(),
            "-ip" => device_ip = args.next().unwrap_or_default(),
            "-start" => start_port = args.next().unwrap_or_default().parse::<u16>().unwrap_or(start_port),
            "-end" => end_port = args.next().unwrap_or_default().parse::<u16>().unwrap_or(end_port),
            "-timeout" => timeout_ms = args.next().unwrap_or_default().parse::<u64>().unwrap_or(timeout_ms),
            "-workers" => workers = args.next().unwrap_or_default().parse::<usize>().unwrap_or(workers),
            _ => {}
        }
    }

    if start_port > end_port {
        eprintln!("invalid port range");
        std::process::exit(2);
    }

    let adb_path = resolve_adb_path(&adb_path).unwrap_or_else(|| adb_path.clone());

    if device_ip.is_empty() {
        match resolve_device_ip(&adb_path) {
            Ok(ip) => device_ip = ip,
            Err(err) => {
                eprintln!("cannot resolve device IP: {err}");
                eprintln!("adb path used: {}", adb_path);
                std::process::exit(1);
            }
        }
    }

    println!("Scanning {} from port {} to {} with timeout={}ms...", device_ip, start_port, end_port, timeout_ms);

    let open_ports = scan_ports(&device_ip, start_port, end_port, timeout_ms, workers);
    if open_ports.is_empty() {
        println!("No open ports found.");
        return;
    }

    println!("Open ports:");
    for port in &open_ports {
        println!("- {}", port);
    }

    let port = open_ports[0];
    let target = format!("{}:{}", device_ip, port);
    if let Some(scrcpy_path) = resolve_scrcpy_path() {
        println!("Launching scrcpy for {}...", target);
        let _ = launch_scrcpy(&scrcpy_path, &target);
    } else {
        println!("scrcpy executable not found, skip launching");
    }
}

fn resolve_device_ip(adb_path: &str) -> Result<String, String> {
    let commands = [
        vec!["shell", "ip", "-f", "inet", "addr", "show", "wlan0"],
        vec!["shell", "ip", "route"],
    ];

    for args in commands {
        match run_adb(adb_path, &args) {
            Ok(output) => {
                if let Some(ip) = parse_ip_from_adb_output(&output) {
                    return Ok(ip);
                }
            }
            Err(err) => {
                eprintln!("adb failed for {:?}: {}", args, err);
            }
        }
    }

    Err("could not infer Wi-Fi IP from adb output".to_string())
}

fn parse_ip_from_adb_output(output: &str) -> Option<String> {
    let re = Regex::new(r"(?:inet|src)\s+(\d{1,3}(?:\.\d{1,3}){3})").unwrap();
    for cap in re.captures_iter(output) {
        let ip = cap.get(1)?.as_str().to_string();
        if ip != "127.0.0.1" {
            return Some(ip);
        }
    }
    None
}

fn run_adb(adb_path: &str, args: &[&str]) -> Result<String, String> {
    let output = Command::new(adb_path)
        .args(args)
        .output()
        .map_err(|e| format!("failed to run {}: {}", adb_path, e))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
        return Err(format!("exit {}: stderr='{}' stdout='{}'", output.status, stderr, stdout));
    }

    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

fn resolve_adb_path(explicit: &str) -> Option<String> {
    if !explicit.is_empty() {
        let explicit_path = PathBuf::from(explicit);
        if explicit_path.exists() {
            return Some(explicit.to_string());
        }
    }

    let mut candidates = Vec::new();
    if let Ok(base_dir) = env::current_dir() {
        candidates.push(base_dir.join("adb.exe"));
        candidates.push(base_dir.join(r"..\scrcpy-win64-v3.3.4\adb.exe"));
        candidates.push(base_dir.join(r"..\scrcpy-win64-v3.3.4\platform-tools\adb.exe"));
        candidates.push(base_dir.join(r"..\platform-tools-latest-windows\platform-tools\adb.exe"));
    }
    candidates.push(PathBuf::from("adb.exe"));
    candidates.push(PathBuf::from("adb"));

    for path in candidates {
        if path.exists() {
            return Some(path.to_string_lossy().into_owned());
        }
    }
    None
}

fn scan_ports(host: &str, start_port: u16, end_port: u16, timeout_ms: u64, workers: usize) -> Vec<u16> {
    let (tx_jobs, rx_jobs) = mpsc::channel::<u16>();
    let (tx_results, rx_results) = mpsc::channel::<u16>();
    let host = Arc::new(host.to_string());
    let rx_jobs = Arc::new(Mutex::new(rx_jobs));

    for _ in 0..workers.min(1000) {
        let host_clone = Arc::clone(&host);
        let tx_results_clone = tx_results.clone();
        let rx_jobs_clone = Arc::clone(&rx_jobs);
        thread::spawn(move || {
            let timeout = Duration::from_millis(timeout_ms);
            while let Ok(port) = rx_jobs_clone.lock().unwrap().recv() {
                if is_open(&host_clone, port, timeout) {
                    let _ = tx_results_clone.send(port);
                }
            }
        });
    }

    let tx_jobs_clone = tx_jobs.clone();
    thread::spawn(move || {
        for port in start_port..=end_port {
            let _ = tx_jobs_clone.send(port);
        }
    });

    drop(tx_jobs);

    let mut open_ports = Vec::new();
    while let Ok(port) = rx_results.recv() {
        open_ports.push(port);
    }

    open_ports.sort();
    open_ports
}

fn is_open(host: &str, port: u16, timeout: Duration) -> bool {
    let addr = format!("{}:{}", host, port);
    if let Ok(socket_addr) = addr.parse::<SocketAddr>() {
        match TcpStream::connect_timeout(&socket_addr, timeout) {
            Ok(_) => true,
            Err(_) => false,
        }
    } else {
        false
    }
}

fn resolve_scrcpy_path() -> Option<String> {
    let base_dir = env::current_dir().ok()?;
    let candidates = [
        base_dir.join("scrcpy.exe"),
        base_dir.join("scrcpy-win64-v3.3.4").join("scrcpy.exe"),
        base_dir.join("scrcpy-win64-v3.3.4").join("scrcpy-console.bat"),
    ];

    for path in candidates {
        if path.exists() {
            return Some(path.to_string_lossy().into_owned());
        }
    }
    None
}

fn launch_scrcpy(scrcpy_path: &str, target: &str) -> io::Result<()> {
    let mut cmd = Command::new(scrcpy_path);
    cmd.arg(format!("--tcpip={}", target));
    cmd.stdout(Stdio::inherit());
    cmd.stderr(Stdio::inherit());
    cmd.spawn()?;
    Ok(())
}
