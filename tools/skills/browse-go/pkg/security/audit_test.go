package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogAudit(t *testing.T) {
	dir := t.TempDir()
	SetAuditLogPath(filepath.Join(dir, "audit.jsonl"))
	defer SetAuditLogPath("")

	ok := LogAudit(AuditRecord{Command: "text", Granted: true})
	if !ok {
		t.Fatal("expected LogAudit to succeed")
	}

	// Verify file exists and contains the record
	data, err := os.ReadFile(auditLogPath)
	if err != nil {
		t.Fatalf("expected audit log to exist: %v", err)
	}
	if !strings.Contains(string(data), `"command":"text"`) {
		t.Errorf("expected audit log to contain command=text, got: %s", string(data))
	}
	if !strings.Contains(string(data), `"granted":true`) {
		t.Errorf("expected audit log to contain granted=true")
	}
}

func TestQueryAudit(t *testing.T) {
	dir := t.TempDir()
	SetAuditLogPath(filepath.Join(dir, "audit.jsonl"))
	defer SetAuditLogPath("")

	// Log several records
	_ = LogAudit(AuditRecord{Command: "text", Verdict: "safe", Scope: "read"})
	_ = LogAudit(AuditRecord{Command: "goto", Verdict: "safe", Scope: "navigate"})
	_ = LogAudit(AuditRecord{Command: "click", Verdict: "warn", Scope: "interact"})
	_ = LogAudit(AuditRecord{Command: "text", Verdict: "block", Scope: "read"})

	// Query by command
	results, err := QueryAudit(QueryAuditOptions{Command: "text", MaxResults: 10})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 text results, got %d", len(results))
	}

	// Query by verdict
	results, err = QueryAudit(QueryAuditOptions{Verdict: "warn", MaxResults: 10})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(results) != 1 || results[0].Command != "click" {
		t.Errorf("expected 1 warn result (click), got %+v", results)
	}

	// Query with max results
	results, err = QueryAudit(QueryAuditOptions{MaxResults: 2})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (max), got %d", len(results))
	}
}

func TestQueryAuditEmpty(t *testing.T) {
	dir := t.TempDir()
	SetAuditLogPath(filepath.Join(dir, "nonexistent.jsonl"))
	defer SetAuditLogPath("")

	results, err := QueryAudit(QueryAuditOptions{})
	if err != nil {
		t.Fatalf("query on empty log should not error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestAuditStats(t *testing.T) {
	dir := t.TempDir()
	SetAuditLogPath(filepath.Join(dir, "audit.jsonl"))
	defer SetAuditLogPath("")

	_ = LogAudit(AuditRecord{Command: "text", Verdict: "safe", Scope: "read"})
	_ = LogAudit(AuditRecord{Command: "goto", Verdict: "safe", Scope: "navigate"})
	_ = LogAudit(AuditRecord{Command: "text", Verdict: "warn", Scope: "read"})

	stats := AuditStats()
	if stats["total"] != 3 {
		t.Errorf("expected total=3, got %v", stats["total"])
	}

	byCommand := stats["byCommand"].(map[string]int)
	if byCommand["text"] != 2 {
		t.Errorf("expected text=2, got %d", byCommand["text"])
	}
	if byCommand["goto"] != 1 {
		t.Errorf("expected goto=1, got %d", byCommand["goto"])
	}

	byVerdict := stats["byVerdict"].(map[string]int)
	if byVerdict["safe"] != 2 {
		t.Errorf("expected safe=2, got %d", byVerdict["safe"])
	}
	if byVerdict["warn"] != 1 {
		t.Errorf("expected warn=1, got %d", byVerdict["warn"])
	}
}

func TestAuditStatsEmpty(t *testing.T) {
	dir := t.TempDir()
	SetAuditLogPath(filepath.Join(dir, "audit.jsonl"))
	defer SetAuditLogPath("")

	stats := AuditStats()
	if stats["total"] != 0 {
		t.Errorf("expected total=0 for empty log, got %v", stats["total"])
	}
}

func TestTruncateAudit(t *testing.T) {
	dir := t.TempDir()
	SetAuditLogPath(filepath.Join(dir, "audit.jsonl"))
	defer SetAuditLogPath("")

	// Log a record
	_ = LogAudit(AuditRecord{Command: "text"})

	// Truncate records older than 1 hour (should keep the record)
	err := TruncateAudit(time.Hour)
	if err != nil {
		t.Fatalf("truncate failed: %v", err)
	}
	results, _ := QueryAudit(QueryAuditOptions{MaxResults: 10})
	if len(results) != 1 {
		t.Errorf("expected 1 record after truncate (1h), got %d", len(results))
	}

	// Truncate records older than 0 (should remove all)
	err = TruncateAudit(0)
	if err != nil {
		t.Fatalf("truncate failed: %v", err)
	}
	results, _ = QueryAudit(QueryAuditOptions{MaxResults: 10})
	if len(results) != 0 {
		t.Errorf("expected 0 records after truncate (0), got %d", len(results))
	}
}

func TestExportAudit(t *testing.T) {
	dir := t.TempDir()
	SetAuditLogPath(filepath.Join(dir, "audit.jsonl"))
	defer SetAuditLogPath("")

	_ = LogAudit(AuditRecord{Command: "text", Verdict: "safe"})
	_ = LogAudit(AuditRecord{Command: "goto", Verdict: "safe"})

	outPath := filepath.Join(dir, "export.jsonl")
	err := ExportAudit(QueryAuditOptions{Command: "text"}, outPath)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("export file not found: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 exported line, got %d", len(lines))
	}
}
