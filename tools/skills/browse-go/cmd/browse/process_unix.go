//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func safeKill(pid int, sig syscall.Signal) error {
	return syscall.Kill(pid, sig)
}

func killTerminalAgent() error {
	return exec.Command("pkill", "-f", "terminal-agent").Run()
}

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func killOrphanedChromium(profileDir string) {
	singletonLock := filepath.Join(profileDir, "SingletonLock")
	target, err := os.Readlink(singletonLock)
	if err != nil {
		return // ENOENT or not a symlink — nothing to do
	}
	// Target format: "hostname-12345" where 12345 is the PID
	parts := strings.Split(target, "-")
	if len(parts) == 0 {
		return
	}
	orphanPid, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil || orphanPid <= 0 {
		return
	}
	if isProcessAlive(orphanPid) {
		_ = safeKill(orphanPid, sigTerm)
		time.Sleep(1 * time.Second)
		if isProcessAlive(orphanPid) {
			_ = safeKill(orphanPid, sigKill)
		}
	}
}
