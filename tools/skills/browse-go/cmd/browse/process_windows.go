//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func safeKill(pid int, sig syscall.Signal) error {
	return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
}

func killTerminalAgent() error {
	return exec.Command("taskkill", "/F", "/IM", "terminal-agent.exe").Run()
}

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func isProcessAlive(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}

func killOrphanedChromium(_ string) {
	// Windows Chromium doesn't use symlink-based SingletonLock.
	// No-op; stale processes are cleaned up by safeKill on restart.
}
