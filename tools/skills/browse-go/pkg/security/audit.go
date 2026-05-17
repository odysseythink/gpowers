package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AuditRecord is a single entry in the security audit log.
// Captures all command executions, not just security events.
type AuditRecord struct {
	TS          string  `json:"ts"`
	SessionID   string  `json:"sessionId,omitempty"`
	ClientID    string  `json:"clientId,omitempty"`
	Command     string  `json:"command"`
	Args        []string `json:"args,omitempty"`
	URLDomain   string  `json:"urlDomain,omitempty"`
	Scope       string  `json:"scope,omitempty"`
	Granted     bool    `json:"granted"`
	RateLimited bool    `json:"rateLimited,omitempty"`
	Verdict     string  `json:"verdict,omitempty"`     // "safe" | "warn" | "block" | ""
	Confidence  float64 `json:"confidence,omitempty"`
	Layer       string  `json:"layer,omitempty"`
	Error       string  `json:"error,omitempty"`
	DurationMs  int64   `json:"durationMs,omitempty"`
}

var (
	auditLogPath string
	auditMu      sync.Mutex
)

func initAuditPath() {
	if auditLogPath == "" {
		auditLogPath = filepath.Join(securityDir, "audit.jsonl")
	}
}

// SetAuditLogPath overrides the default audit log path.
func SetAuditLogPath(path string) {
	auditMu.Lock()
	defer auditMu.Unlock()
	auditLogPath = path
}

// LogAudit appends a record to the local audit log.
// Never throws — logging failure should not break the session.
func LogAudit(record AuditRecord) bool {
	initAuditPath()
	record.TS = time.Now().UTC().Format(time.RFC3339)
	_ = os.MkdirAll(securityDir, 0750)
	rotateAuditIfNeeded()

	line, err := json.Marshal(record)
	if err != nil {
		return false
	}

	auditMu.Lock()
	defer auditMu.Unlock()

	f, err := os.OpenFile(auditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err == nil
}

// LogCommandStart records the start of a command execution.
// Returns the start time for calculating duration in LogCommandEnd.
func LogCommandStart(command string, args []string, scope string, clientID string) time.Time {
	return time.Now()
}

// LogCommandEnd records the completion of a command execution.
func LogCommandEnd(start time.Time, command string, args []string, urlDomain string, scope string, clientID string, granted bool, rateLimited bool, result *SecurityResult, execErr error) {
	record := AuditRecord{
		Command:     command,
		Args:        args,
		URLDomain:   urlDomain,
		Scope:       scope,
		ClientID:    clientID,
		Granted:     granted,
		RateLimited: rateLimited,
		DurationMs:  time.Since(start).Milliseconds(),
	}
	if result != nil {
		record.Verdict = string(result.Verdict)
		record.Confidence = result.Confidence
		record.Layer = result.Reason
	}
	if execErr != nil {
		record.Error = execErr.Error()
	}
	_ = LogAudit(record)
}

// QueryAuditOptions controls filtering of audit log queries.
type QueryAuditOptions struct {
	Command    string    // filter by command name
	ClientID   string    // filter by client ID
	URLDomain  string    // filter by URL domain
	From       time.Time // inclusive
	To         time.Time // inclusive
	Verdict    string    // filter by verdict
	MaxResults int       // default 100, 0 = all
}

// QueryAudit reads audit records matching the given criteria.
func QueryAudit(opts QueryAuditOptions) ([]AuditRecord, error) {
	initAuditPath()
	if opts.MaxResults == 0 {
		opts.MaxResults = 100
	}

	data, err := os.ReadFile(auditLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []AuditRecord
	lines := strings.Split(string(data), "\n")
	// Read in reverse (newest first)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var rec AuditRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		if !matchAudit(rec, opts) {
			continue
		}
		results = append(results, rec)
		if len(results) >= opts.MaxResults {
			break
		}
	}
	return results, nil
}

func matchAudit(rec AuditRecord, opts QueryAuditOptions) bool {
	if opts.Command != "" && rec.Command != opts.Command {
		return false
	}
	if opts.ClientID != "" && rec.ClientID != opts.ClientID {
		return false
	}
	if opts.URLDomain != "" && rec.URLDomain != opts.URLDomain {
		return false
	}
	if opts.Verdict != "" && rec.Verdict != opts.Verdict {
		return false
	}
	if !opts.From.IsZero() {
		ts, _ := time.Parse(time.RFC3339, rec.TS)
		if ts.Before(opts.From) {
			return false
		}
	}
	if !opts.To.IsZero() {
		ts, _ := time.Parse(time.RFC3339, rec.TS)
		if ts.After(opts.To) {
			return false
		}
	}
	return true
}

// ExportAudit writes audit records matching opts to a JSONL file.
func ExportAudit(opts QueryAuditOptions, outPath string) error {
	records, err := QueryAudit(opts)
	if err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, rec := range records {
		line, _ := json.Marshal(rec)
		_, _ = f.Write(append(line, '\n'))
	}
	return nil
}

// Stats returns aggregate statistics from the audit log.
func AuditStats() map[string]interface{} {
	initAuditPath()
	data, err := os.ReadFile(auditLogPath)
	if err != nil {
		return map[string]interface{}{"total": 0}
	}

	total := 0
	byCommand := make(map[string]int)
	byVerdict := make(map[string]int)
	byScope := make(map[string]int)
	var firstTS, lastTS time.Time

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec AuditRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		total++
		byCommand[rec.Command]++
		if rec.Verdict != "" {
			byVerdict[rec.Verdict]++
		}
		if rec.Scope != "" {
			byScope[rec.Scope]++
		}
		ts, _ := time.Parse(time.RFC3339, rec.TS)
		if !ts.IsZero() {
			if firstTS.IsZero() || ts.Before(firstTS) {
				firstTS = ts
			}
			if ts.After(lastTS) {
				lastTS = ts
			}
		}
	}

	return map[string]interface{}{
		"total":      total,
		"firstEvent": firstTS.Format(time.RFC3339),
		"lastEvent":  lastTS.Format(time.RFC3339),
		"byCommand":  byCommand,
		"byVerdict":  byVerdict,
		"byScope":    byScope,
	}
}

const (
	maxAuditBytes       = 50 * 1024 * 1024 // 50MB rotate threshold
	maxAuditGenerations = 3
)

func rotateAuditIfNeeded() {
	info, err := os.Stat(auditLogPath)
	if err != nil || info.Size() < maxAuditBytes {
		return
	}
	for i := maxAuditGenerations - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", auditLogPath, i)
		dst := fmt.Sprintf("%s.%d", auditLogPath, i+1)
		_ = os.Rename(src, dst)
	}
	_ = os.Rename(auditLogPath, auditLogPath+".1")
}

// TruncateAudit removes all audit log records older than the given duration.
func TruncateAudit(olderThan time.Duration) error {
	initAuditPath()
	cutoff := time.Now().UTC().Add(-olderThan)
	data, err := os.ReadFile(auditLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec AuditRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		ts, _ := time.Parse(time.RFC3339, rec.TS)
		if !ts.IsZero() && ts.After(cutoff) {
			kept = append(kept, line)
		}
	}

	if len(kept) == 0 {
		_ = os.Remove(auditLogPath)
		return nil
	}

	// Atomic write
	tmp := auditLogPath + ".tmp." + strconv.Itoa(os.Getpid())
	if err := os.WriteFile(tmp, []byte(strings.Join(kept, "\n")+"\n"), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, auditLogPath)
}
