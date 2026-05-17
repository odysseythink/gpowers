// Package telemetry provides lightweight, fire-and-forget local event logging
// for DX analytics. All data stays on-disk; no remote transmission.
//
// Events are appended to ~/.gstack/analytics/browse-telemetry.jsonl as
// newline-delimited JSON. Telemetry is disabled when GSTACK_TELEMETRY_OFF=1.
//
// Usage:
//
//	telemetry.Log(telemetry.Event{Event: "domain_skill_saved", Host: host})
package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is a telemetry entry. Use a map or anonymous struct for extra fields.
type Event map[string]interface{}

var (
	disabled     bool
	disabledOnce sync.Once
	mu           sync.Mutex
)

func isDisabled() bool {
	disabledOnce.Do(func() {
		disabled = os.Getenv("GSTACK_TELEMETRY_OFF") == "1"
	})
	return disabled
}

// Log writes a telemetry event fire-and-forget. Never blocks or panics.
func Log(payload Event) {
	if isDisabled() {
		return
	}
	if payload == nil {
		payload = Event{}
	}
	if payload["ts"] == nil {
		payload["ts"] = time.Now().UTC().Format(time.RFC3339)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	data = append(data, '\n')

	go func() {
		mu.Lock()
		defer mu.Unlock()

		dir := analyticsDir()
		_ = os.MkdirAll(dir, 0755)
		_ = appendFile(filepath.Join(dir, "browse-telemetry.jsonl"), data)
	}()
}

func analyticsDir() string {
	home := os.Getenv("GSTACK_HOME")
	if home == "" {
		homeDir, _ := os.UserHomeDir()
		home = filepath.Join(homeDir, ".gstack")
	}
	return filepath.Join(home, "analytics")
}

func appendFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// ResetCache clears the disabled-once cache (test-only).
func ResetCache() {
	disabledOnce = sync.Once{}
	disabled = false
}
