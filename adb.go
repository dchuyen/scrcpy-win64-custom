package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

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
