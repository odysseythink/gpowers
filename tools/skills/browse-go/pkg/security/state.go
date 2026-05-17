package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionState holds cross-process security state.
// Written by the server process, read by any process that needs status.
type SessionState struct {
	SessionID        string            `json:"sessionId"`
	Canary           string            `json:"canary"`
	WarnedDomains    []string          `json:"warnedDomains"`
	ClassifierStatus map[string]string `json:"classifierStatus"`
	LastUpdated      string            `json:"lastUpdated"`
}

var (
	securityDir string
	stateFile   string
)

func init() {
	home, _ := os.UserHomeDir()
	securityDir = filepath.Join(home, ".gstack", "security")
	stateFile = filepath.Join(securityDir, "session-state.json")
}

// WriteSessionState atomically writes session state (temp + rename pattern).
func WriteSessionState(state SessionState) error {
	state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(securityDir, 0750); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", stateFile, os.Getpid())
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, stateFile)
}

// ReadSessionState reads the current session state, or returns nil if not present.
func ReadSessionState() *SessionState {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil
	}
	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	return &state
}

// DecisionRecord captures a user's security decision for a tab.
type DecisionRecord struct {
	TabID    int       `json:"tabId"`
	Decision string    `json:"decision"` // "allow" | "block"
	TS       string    `json:"ts"`
	Reason   string    `json:"reason,omitempty"`
}

var decisionsDir string

func init() {
	decisionsDir = filepath.Join(securityDir, "decisions")
}

func decisionFileForTab(tabID int) string {
	return filepath.Join(decisionsDir, fmt.Sprintf("tab-%d.json", tabID))
}

// WriteDecision atomically writes a security decision.
func WriteDecision(record DecisionRecord) error {
	record.TS = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(decisionsDir, 0750); err != nil {
		return err
	}
	file := decisionFileForTab(record.TabID)
	tmp := fmt.Sprintf("%s.tmp.%d", file, os.Getpid())
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// ReadDecision reads the decision for a tab, or nil if none.
func ReadDecision(tabID int) *DecisionRecord {
	data, err := os.ReadFile(decisionFileForTab(tabID))
	if err != nil {
		return nil
	}
	var record DecisionRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil
	}
	return &record
}

// ClearDecision removes a tab's decision file.
func ClearDecision(tabID int) {
	_ = os.Remove(decisionFileForTab(tabID))
}

// GetStatus returns the overall security status for the shield icon.
func GetStatus() *StatusDetail {
	state := ReadSessionState()
	layers := map[string]string{
		"testsavant": "off",
		"transcript": "off",
		"canary":     "off",
	}
	if state != nil {
		for k, v := range state.ClassifierStatus {
			layers[k] = v
		}
		if state.Canary != "" {
			layers["canary"] = "ok"
		}
	}

	var status SecurityStatus
	if layers["testsavant"] == "ok" && layers["transcript"] == "ok" && layers["canary"] == "ok" {
		status = StatusProtected
	} else if layers["testsavant"] == "off" && layers["canary"] == "off" {
		status = StatusInactive
	} else {
		status = StatusDegraded
	}

	lastUpdated := time.Now().UTC().Format(time.RFC3339)
	if state != nil && state.LastUpdated != "" {
		lastUpdated = state.LastUpdated
	}

	return &StatusDetail{
		Status:      status,
		Layers:      layers,
		LastUpdated: lastUpdated,
	}
}

// ExtractDomain extracts the hostname from a URL. Never logs path or query.
func ExtractDomain(urlStr string) string {
	// Simple extraction without full URL parsing to avoid errors on malformed input
	if idx := strings.Index(urlStr, "://"); idx != -1 {
		urlStr = urlStr[idx+3:]
	}
	if idx := strings.Index(urlStr, "/"); idx != -1 {
		urlStr = urlStr[:idx]
	}
	if idx := strings.Index(urlStr, ":"); idx != -1 {
		urlStr = urlStr[:idx]
	}
	return urlStr
}
