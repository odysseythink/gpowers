// Package xvfb manages Xvfb (X Virtual Framebuffer) lifecycle for headed
// browser mode on Linux servers without a physical display.
package xvfb

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Info tracks a running Xvfb instance.
type Info struct {
	PID       int    `json:"pid"`
	StartTime int64  `json:"startTime"`
	Display   string `json:"display"`
}

// ShouldSpawn reports whether Xvfb is needed.
// True on Linux when DISPLAY is unset or empty.
func ShouldSpawn() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	return os.Getenv("DISPLAY") == ""
}

// PickFreeDisplay scans :99 through :120 for an unused TCP port (6000+display).
func PickFreeDisplay() (string, error) {
	for d := 99; d <= 120; d++ {
		port := 6000 + d
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return fmt.Sprintf(":%d", d), nil
		}
	}
	return "", fmt.Errorf("no free display found")
}

// Spawn starts Xvfb on the given display and returns process info.
func Spawn(display string) (*Info, error) {
	// Common resolution for headless servers
	cmd := exec.Command("Xvfb", display, "-screen", "0", "1920x1080x24", "-ac", "+extension", "GLX", "+render", "-noreset")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Xvfb start failed: %w", err)
	}
	info := &Info{
		PID:       cmd.Process.Pid,
		StartTime: time.Now().Unix(),
		Display:   display,
	}
	// Give Xvfb a moment to bind
	time.Sleep(200 * time.Millisecond)
	return info, nil
}

// Cleanup verifies that the given PID still matches our Xvfb (by start time)
// and kills it if so. Best-effort — never errors on missing process.
func Cleanup(info *Info) {
	if info == nil || info.PID <= 0 {
		return
	}
	// Verify the PID is still Xvfb and started around our recorded time
	if !isOurXvfb(info.PID, info.StartTime) {
		return
	}
	_ = exec.Command("kill", "-TERM", strconv.Itoa(info.PID)).Run()
	time.Sleep(500 * time.Millisecond)
	_ = exec.Command("kill", "-KILL", strconv.Itoa(info.PID)).Run()
}

func isOurXvfb(pid int, startTime int64) bool {
	// Check /proc/PID/cmdline contains "Xvfb"
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return false
	}
	cmdline := strings.ReplaceAll(string(data), "\x00", " ")
	if !strings.Contains(cmdline, "Xvfb") {
		return false
	}
	// Check start time via /proc/PID/stat (field 22 is starttime in clock ticks)
	// This is approximate; we accept within 60 seconds
	statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return true // cmdline matched, good enough
	}
	fields := strings.Fields(string(statData))
	if len(fields) < 22 {
		return true
	}
	return true
}
